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
	"github.com/couchbase/query/value"
)

type IntersectScan struct {
	base
	plan    *plan.IntersectScan
	scans   []Operator
	values  map[string]value.AnnotatedValue
	bits    map[string]int64
	sent    int64
	channel *Channel
}

func NewIntersectScan(plan *plan.IntersectScan, context *Context, scans []Operator) *IntersectScan {
	rv := &IntersectScan{
		plan:  plan,
		scans: scans,
	}

	newBase(&rv.base, context)
	rv.trackChildren(len(scans))
	rv.output = rv
	return rv
}

func (this *IntersectScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIntersectScan(this)
}

func (this *IntersectScan) Copy() Operator {
	scans := _INDEX_SCAN_POOL.Get()

	for _, s := range this.scans {
		scans = append(scans, s.Copy())
	}

	rv := &IntersectScan{
		plan:  this.plan,
		scans: scans,
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *IntersectScan) PlanOp() plan.Operator {
	return this.plan
}

func (this *IntersectScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		defer this.notify() // Notify that I have stopped

		if !active || !context.assert(len(this.scans) != 0, "Intersect scan has no scans") {
			return
		}

		defer func() {
			this.values = nil
			this.bits = nil
		}()

		pipelineCap := int(context.GetPipelineCap())
		if pipelineCap <= _INDEX_VALUE_POOL.Size() {
			this.values = _INDEX_VALUE_POOL.Get()
			this.bits = _INDEX_BIT_POOL.Get()

			defer func() {
				_INDEX_VALUE_POOL.Put(this.values)
				_INDEX_BIT_POOL.Put(this.bits)
			}()
		} else {
			this.values = make(map[string]value.AnnotatedValue, pipelineCap)
			this.bits = make(map[string]int64, pipelineCap)
		}

		fullBits := int64(0)
		for i, scan := range this.scans {
			scan.SetBit(uint8(i))
			fullBits |= int64(0x01) << uint8(i)
		}

		this.channel = NewChannel(context)
		this.SetInput(this.channel)

		for _, scan := range this.scans {
			scan.SetParent(this)
			scan.SetOutput(this.channel)
			this.fork(scan, context, parent)
		}

		limit := evalLimitOffset(this.plan.Limit(), parent, int64(-1), this.plan.Covering(), context)
		n := len(this.scans)
		nscans := len(this.scans)
		childBits := int64(0)
		stopped := false
		ok := true

	loop:
		for ok {
			item, childBit, cont := this.getItemChildren()
			if cont {
				if childBit >= 0 {

					// MB-22321 terminate when first child terminates
					if n == nscans {
						sendChildren(this.plan, this.scans...)
						childBits |= int64(0x01) << uint(childBit)
					}
					n--

					// now that all children are gone, flag that there's
					// no more values coming in
					if n == 0 {
						this.channel.close(context)
					}
				} else if item != nil {
					this.addInDocs(1)
					ok = this.processKey(item, context, fullBits, limit)
					stopped = this.stopped
				} else {
					break loop
				}
			} else {
				stopped = true

				// if not done already, stop children, wait and clean up
				if n == nscans {
					sendChildren(this.plan, this.scans...)
				}
				if n > 0 {
					this.childrenWaitNoStop(this.scans...)
					this.channel.close(context)
				}
				break loop
			}
		}

		if !stopped && ok && childBits != 0 && (limit <= 0 || this.sent < limit) {
			this.sendItems(childBits)
		}
	})
}

func (this *IntersectScan) processKey(item value.AnnotatedValue,
	context *Context, fullBits, limit int64) bool {

	key, ok := this.getDocumentKey(item, context)
	if !ok {
		return false
	}

	bits := this.bits[key]
	if bits == 0 {
		this.values[key] = item
	} else {
		mergeSearchMeta(this.values[key], item)
	}

	bits |= int64(0x01) << item.Bit()

	if (bits&fullBits)^fullBits == 0 {
		item = this.values[key]
		delete(this.values, key)
		delete(this.bits, key)

		if limit > 0 {
			this.sent++
		}

		item.SetBit(this.bit)
		return this.sendItem(item) && (limit <= 0 || this.sent < limit)
	}

	this.bits[key] = bits
	return true
}

func (this *IntersectScan) sendItems(childBits int64) {
	if childBits == 0 {
		return
	}

	for key, bits := range this.bits {
		if (bits&childBits)^childBits == 0 {
			item := this.values[key]
			item.SetBit(this.bit)
			if !this.sendItem(item) {
				return
			}
		}
	}
}

func (this *IntersectScan) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
		r["scans"] = this.scans
	})
	return json.Marshal(r)
}

func (this *IntersectScan) accrueTimes(o Operator) {
	if baseAccrueTimes(this, o) {
		return
	}
	copy, _ := o.(*IntersectScan)
	childrenAccrueTimes(this.scans, copy.scans)
}

func (this *IntersectScan) SendAction(action opAction) {
	if this.baseSendAction(action) {
		scans := this.scans
		for _, scan := range scans {
			if scan != nil {
				scan.SendAction(action)
			}
			if this.scans == nil {
				break
			}
		}
	}
}

func (this *IntersectScan) reopen(context *Context) bool {
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

func (this *IntersectScan) Done() {
	this.baseDone()
	for s, scan := range this.scans {
		this.scans[s] = nil
		scan.Done()
	}
	_INDEX_SCAN_POOL.Put(this.scans)
	this.scans = nil
	channel := this.channel
	this.channel = nil
	if channel != nil {
		channel.Done()
	}
}

func mergeSearchMeta(dest, src value.AnnotatedValue) {
	srcMeta := src.GetAttachment("smeta")
	if srcMeta == nil {
		return
	}

	destMeta := dest.GetAttachment("smeta")
	if destMeta == nil {
		dest.SetAttachment("smeta", srcMeta)
	} else {
		s := srcMeta.(map[string]interface{})
		d := destMeta.(map[string]interface{})
		for n, v := range s {
			d[n] = v
		}
		dest.SetAttachment("smeta", d)
	}
}

func sendChildren(op plan.SecondaryScan, children ...Operator) {
	if op.IsUnderNL() {
		pauseChildren(children...)
	} else {
		notifyChildren(children...)
	}
}
