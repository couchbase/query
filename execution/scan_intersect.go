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
	"fmt"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type IntersectScan struct {
	base
	plan         *plan.IntersectScan
	scans        []Operator
	counts       map[string]int
	values       map[string]value.AnnotatedValue
	bits         map[string]int64
	childChannel StopChannel
	sent         int64
}

func NewIntersectScan(plan *plan.IntersectScan, scans []Operator) *IntersectScan {
	rv := &IntersectScan{
		base:         newBase(),
		plan:         plan,
		scans:        scans,
		childChannel: make(StopChannel, len(scans)),
	}

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

	return &IntersectScan{
		base:         this.base.copy(),
		plan:         this.plan,
		scans:        scans,
		childChannel: make(StopChannel, len(scans)),
	}
}

func (this *IntersectScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover()       // Recover from any panic
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		this.counts = _INDEX_COUNT_POOL.Get()
		defer func() {
			_INDEX_SCAN_POOL.Put(this.scans)
			this.scans = nil
			_INDEX_COUNT_POOL.Put(this.counts)
			this.counts = nil
		}()

		if len(this.scans) <= 64 {
			this.bits = _INDEX_BIT_POOL.Get()
			this.values = _INDEX_VALUE_POOL.Get()
			defer func() {
				_INDEX_BIT_POOL.Put(this.bits)
				_INDEX_VALUE_POOL.Put(this.values)
				this.bits = nil
				this.values = nil
			}()

			for i, scan := range this.scans {
				scan.SetBit(uint8(i))
			}
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
			select {
			case <-this.stopChannel:
				stopped = true
				break loop
			default:
			}

			select {
			case childBit = <-this.childChannel:
				if n == nscans && (this.bits != nil || len(this.counts) == 0) {
					notifyChildren(this.scans...)
				}
				n--
				childBits |= int64(0x01) << uint(childBit)
			default:
			}

			select {
			case item, ok = <-channel.ItemChannel():
				if ok {
					ok = this.processKey(item, context, limit)
				}
			case childBit = <-this.childChannel:
				if n == nscans && (this.bits != nil || len(this.counts) == 0) {
					notifyChildren(this.scans...)
				}
				n--
				childBits |= int64(0x01) << uint(childBit)
			case <-this.stopChannel:
				stopped = true
				break loop
			default:
				if n == 0 || (n < nscans && (this.bits != nil || len(this.counts) == 0)) {
					break loop
				}
			}
		}

		// Await children
		notifyChildren(this.scans...)
		for ; n > 0; n-- {
			<-this.childChannel
		}

		if !stopped && len(this.bits) > 0 && (limit <= 0 || this.sent < limit) {
			this.sendItems(childBits)
		}
	})
}

func (this *IntersectScan) ChildChannel() StopChannel {
	return this.childChannel
}

func (this *IntersectScan) processKey(item value.AnnotatedValue,
	context *Context, limit int64) bool {

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

	count := this.counts[key]

	if count+1 == len(this.scans) {
		delete(this.counts, key)
		if this.values != nil {
			delete(this.values, key)
			delete(this.bits, key)
		}

		if limit > 0 {
			this.sent++
		}

		item.SetBit(this.bit)
		return this.sendItem(item) && (limit <= 0 || this.sent < limit)
	}

	this.counts[key] = count + 1

	if this.values != nil {
		this.bits[key] |= int64(0x01) << item.Bit()
		if count == 0 {
			this.values[key] = item
		}
	}

	return true
}

func (this *IntersectScan) sendItems(childBits int64) {
	for key, bits := range this.bits {
		if ((bits & childBits) ^ childBits) == 0 {
			val := this.values[key]
			val.SetBit(this.bit)
			if !this.sendItem(val) {
				return
			}
		}
	}
}
