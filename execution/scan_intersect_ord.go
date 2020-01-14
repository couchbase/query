//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execution

import (
	"encoding/json"

	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type OrderedIntersectScan struct {
	base
	plan      *plan.OrderedIntersectScan
	scans     []Operator
	values    map[string]value.AnnotatedValue
	bits      map[string]int64
	queue     *util.Queue
	sent      int64
	fullCount int64
}

func NewOrderedIntersectScan(plan *plan.OrderedIntersectScan, context *Context, scans []Operator) *OrderedIntersectScan {
	rv := &OrderedIntersectScan{
		plan:  plan,
		scans: scans,
	}

	newBase(&rv.base, context)
	rv.trackChildren(len(scans))
	rv.output = rv
	return rv
}

func (this *OrderedIntersectScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitOrderedIntersectScan(this)
}

func (this *OrderedIntersectScan) Copy() Operator {
	scans := _INDEX_SCAN_POOL.Get()

	for _, s := range this.scans {
		scans = append(scans, s.Copy())
	}

	rv := &OrderedIntersectScan{
		plan:  this.plan,
		scans: scans,
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *OrderedIntersectScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		defer this.notify() // Notify that I have stopped

		defer func() {
			this.values = nil
			this.bits = nil
			this.queue = nil
		}()

		if !active || !context.assert(len(this.scans) != 0, "Ordered Intersect Scan has no scans") {
			return
		}
		pipelineCap := int(context.GetPipelineCap())
		if pipelineCap <= _INDEX_VALUE_POOL.Size() {
			this.values = _INDEX_VALUE_POOL.Get()
			this.bits = _INDEX_BIT_POOL.Get()
			this.queue = _QUEUE_POOL.Get()

			defer func() {
				_INDEX_VALUE_POOL.Put(this.values)
				_INDEX_BIT_POOL.Put(this.bits)
				_QUEUE_POOL.Put(this.queue)
			}()
		} else {
			this.values = make(map[string]value.AnnotatedValue, pipelineCap)
			this.bits = make(map[string]int64, pipelineCap)
			this.queue = util.NewQueue(pipelineCap)
		}

		fullBits := int64(0)
		for i, scan := range this.scans {
			scan.SetBit(uint8(i))
			fullBits |= int64(0x01) << uint8(i)
		}

		channel := NewChannel(context)
		defer channel.Done()
		this.SetInput(channel)

		for _, scan := range this.scans {
			scan.SetParent(this)
			scan.SetOutput(channel)
			go scan.RunOnce(context, parent)
		}

		limit := int64(-1)
		n := len(this.scans)
		nscans := len(this.scans)
		childBits := int64(0)
		firstStopped := false
		needProcessing := -1
		stopped := false
		ok := true

	loop:
		for ok {
			item, childBit, cont := this.getItemChildren()
			if cont {
				if childBit >= 0 {

					// MB-22321 we stop when the first scan finishes
					if childBit == 0 || n == nscans {
						if nscans > 1 {
							notifyChildren(this.scans[1:]...)
						}
						childBits |= int64(0x01) << uint(childBit)
					}
					if childBit == 0 {
						firstStopped = true
					}
					n--

					// MB-31336: if only the first scan remains, process the existing keys
					// and then stop it
					if n == 1 && !firstStopped {
						needProcessing = this.queuedItems()
					}

					// now that all children are gone, flag that there's
					// no more values coming in
					if n == 0 {
						channel.close(context)
					}
				} else if item != nil {
					this.addInDocs(1)

					ok = this.processKey(item, context, fullBits, fullBits, limit, false)
					stopped = this.stopped

					if ok {

						// MB-31336: now that there are no keys that need processing
						// terminate first scan if there are no more keys in the queue
						if needProcessing > 0 {
							needProcessing--
						}
						if needProcessing == 0 && !firstStopped && len(this.bits) == 0 {
							notifyChildren(this.scans[0])
							firstStopped = true
						}
						if limit > 0 && this.fullCount >= limit {
							childBits |= int64(0x01)
							break loop
						}
					}
				} else {
					break loop
				}
			} else {
				stopped = true

				// if not done already, stop children, wait and clean up
				if n == nscans {
					notifyChildren(this.scans...)
				}
				if n > 0 {
					notifyChildren(this.scans[0])
					this.childrenWaitNoStop(n)
					channel.close(context)
				}
				break loop
			}
		}

		if !stopped && (limit <= 0 || this.sent < limit) {
			this.processQueue(fullBits, childBits, limit, true)
		}
	})
}

func (this *OrderedIntersectScan) processKey(item value.AnnotatedValue,
	context *Context, fullBits, sendBits, limit int64, finalScan bool) bool {

	key, ok := this.getDocumentKey(item, context)
	if !ok {
		return false
	}

	bit := item.Bit()
	bits, found := this.bits[key]

	if !found || bit == 0 {
		this.values[key] = item
	} else {
		mergeSearchMeta(this.values[key], item)
	}

	if bit == 0 {
		this.queue.Add(key)
	}

	this.bits[key] = bits | (int64(01) << bit)
	if limit > 0 && ((this.bits[key]&fullBits)^fullBits) == 0 {
		this.fullCount++
	}

	return this.processQueue(fullBits, sendBits, limit, finalScan)
}

func (this *OrderedIntersectScan) processQueue(fullBits, sendBits, limit int64,
	final bool) bool {

	queue := this.queue
	for next := queue.Peek(); next != nil; next = queue.Peek() {
		key := next.(string)
		bits := this.bits[key]
		full := false

		if limit > 0 && ((bits&fullBits)^fullBits) == 0 {
			this.sent++
			full = true
		}

		if full || ((bits&sendBits)^sendBits) == 0 {
			item := this.values[key]
			queue.Remove()
			delete(this.values, key)
			delete(this.bits, key)

			item.SetBit(this.bit)
			if !this.sendItem(item) {
				return false
			}

			if limit > 0 && this.sent >= limit {
				break
			}
		} else if final {
			queue.Remove()
			delete(this.values, key)
			delete(this.bits, key)
		} else {
			break
		}
	}

	return true
}

func (this *OrderedIntersectScan) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
		r["scans"] = this.scans
	})
	return json.Marshal(r)
}

func (this *OrderedIntersectScan) accrueTimes(o Operator) {
	if baseAccrueTimes(this, o) {
		return
	}
	copy, _ := o.(*OrderedIntersectScan)
	childrenAccrueTimes(this.scans, copy.scans)
}

func (this *OrderedIntersectScan) SendStop() {
	this.baseSendStop()
	for _, scan := range this.scans {
		if scan != nil {
			scan.SendStop()
		}
	}
}

func (this *OrderedIntersectScan) reopen(context *Context) bool {
	rv := this.baseReopen(context)
	if rv {
		for _, scan := range this.scans {
			if !scan.reopen(context) {
				return false
			}
		}
	}
	return rv
}

func (this *OrderedIntersectScan) Done() {
	this.baseDone()
	for s, scan := range this.scans {
		this.scans[s] = nil
		scan.Done()
	}
	_INDEX_SCAN_POOL.Put(this.scans)
	this.scans = nil
}

var _QUEUE_POOL = util.NewQueuePool(1024)
