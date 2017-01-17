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
	"fmt"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type OrderedIntersectScan struct {
	base
	plan         *plan.OrderedIntersectScan
	scans        []Operator
	values       map[string]value.AnnotatedValue
	bits         map[string]int64
	queue        *util.Queue
	childChannel StopChannel
	sent         int64
}

func NewOrderedIntersectScan(plan *plan.OrderedIntersectScan, scans []Operator) *OrderedIntersectScan {
	rv := &OrderedIntersectScan{
		base:         newBase(),
		plan:         plan,
		scans:        scans,
		childChannel: make(StopChannel, len(scans)),
	}

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

	return &OrderedIntersectScan{
		base:         this.base.copy(),
		plan:         this.plan,
		scans:        scans,
		childChannel: make(StopChannel, len(scans)),
	}
}

func (this *OrderedIntersectScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover() // Recover from any panic
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		this.values = _INDEX_VALUE_POOL.Get()
		this.bits = _INDEX_BIT_POOL.Get()
		this.queue = _QUEUE_POOL.Get()
		defer func() {
			_INDEX_VALUE_POOL.Put(this.values)
			_INDEX_BIT_POOL.Put(this.bits)
			_QUEUE_POOL.Put(this.queue)
			this.values = nil
			this.bits = nil
			this.queue = nil
		}()

		fullBits := int64(0)
		for i, scan := range this.scans {
			scan.SetBit(uint8(i))
			fullBits |= int64(0x01) << uint8(i)
		}

		channel := NewChannel()
		defer channel.Close()

		for _, scan := range this.scans {
			scan.SetParent(this)
			scan.SetOutput(channel)
			go scan.RunOnce(context, parent)
		}

		var item value.AnnotatedValue
		limit := getLimit(this.plan.Limit(), this.plan.Covering(), context)
		n := len(this.scans)
		nscans := len(this.scans)
		stopped := false
		ok := true
		childBit := 0
		childBits := int64(0)

	loop:
		for ok {
			this.switchPhase(_CHANTIME)
			select {
			case <-this.stopChannel:
				stopped = true
				break loop
			default:
			}

			select {
			case childBit = <-this.childChannel:
				if childBit == 0 || len(this.bits) == 0 {
					notifyChildren(this.scans...)
				}
				n--
				childBits |= int64(0x01) << uint(childBit)
			default:
			}

			select {
			case item, ok = <-channel.ItemChannel():
				this.switchPhase(_EXECTIME)
				if ok {
					this.addInDocs(1)
					ok = this.processKey(item, context, fullBits, limit)
				}
			case childBit = <-this.childChannel:
				if childBit == 0 || len(this.bits) == 0 {
					notifyChildren(this.scans...)
				}
				n--
				childBits |= int64(0x01) << uint(childBit)
			case <-this.stopChannel:
				stopped = true
				break loop
			default:
				if n == 0 || (n < nscans && ((childBits&0x01) != 0 || len(this.bits) == 0)) {
					break loop
				}
			}
		}

		// Await children
		this.switchPhase(_CHANTIME)
		notifyChildren(this.scans...)
		for ; n > 0; n-- {
			<-this.childChannel
		}

		if !stopped && (limit <= 0 || this.sent < limit) {
			this.processQueue(fullBits, childBits, limit, true)
		}
	})
}

func (this *OrderedIntersectScan) ChildChannel() StopChannel {
	return this.childChannel
}

func (this *OrderedIntersectScan) processKey(item value.AnnotatedValue,
	context *Context, fullBits, limit int64) bool {

	m := item.GetAttachment("meta")
	meta, ok := m.(map[string]interface{})
	if !ok {
		context.Error(errors.NewInvalidValueError(
			fmt.Sprintf("Missing or invalid meta %v of type %T.", m, m)))
		return false
	}

	k := meta["id"]
	key, ok := k.(string)
	if !ok {
		context.Error(errors.NewInvalidValueError(
			fmt.Sprintf("Missing or invalid primary key %v of type %T.", k, k)))
		return false
	}

	if _, ok := this.values[key]; !ok {
		this.values[key] = item
	}

	if item.Bit() == 0 {
		this.queue.Add(key)
	}

	this.bits[key] |= int64(01) << item.Bit()
	return this.processQueue(fullBits, fullBits, limit, false)
}

func (this *OrderedIntersectScan) processQueue(fullBits, sendBits, limit int64, all bool) bool {
	queue := this.queue
	for next := queue.Peek(); next != nil; next = queue.Peek() {
		key := next.(string)
		bits := this.bits[key]
		full := false

		if limit > 0 && ((bits&fullBits)^fullBits) == 0 {
			this.sent++
			if this.sent > limit {
				return false
			}
			full = true
		}

		if full || ((bits&sendBits)^sendBits) == 0 {
			item := this.values[key]
			queue.Remove()

			if !all {
				delete(this.values, key)
				delete(this.bits, key)
			}

			if !this.sendItem(item) {
				return false
			}
		} else if all {
			queue.Remove()
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

func (this *OrderedIntersectScan) Done() {
	for s, scan := range this.scans {
		scan.Done()
		this.scans[s] = nil
	}
	_INDEX_SCAN_POOL.Put(this.scans)
	this.scans = nil
}

var _QUEUE_POOL = util.NewQueuePool(1024)
