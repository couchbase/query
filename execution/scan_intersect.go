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
	"github.com/couchbase/query/value"
)

type IntersectScan struct {
	base
	plan         *plan.IntersectScan
	scans        []Operator
	counts       map[string]int
	values       map[string]value.AnnotatedValue
	childChannel StopChannel
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
		this.values = _INDEX_VALUE_POOL.Get()

		defer func() {
			_INDEX_COUNT_POOL.Put(this.counts)
			_INDEX_VALUE_POOL.Put(this.values)
			this.counts = nil
			this.values = nil
		}()

		channel := NewChannel()
		defer channel.Close()

		for _, scan := range this.scans {
			scan.SetParent(this)
			scan.SetOutput(channel)
			go scan.RunOnce(context, parent)
		}

		var item value.AnnotatedValue
		n := len(this.scans)
		nscans := len(this.scans)
		ok := true

	loop:
		for ok {
			select {
			case <-this.stopChannel:
				break loop
			default:
			}

			select {
			case item, ok = <-channel.ItemChannel():
				if ok {
					ok = this.processKey(item, context)
				}
			case <-this.childChannel:
				n--
			case <-this.stopChannel:
				break loop
			default:
				if n == 0 || (n < nscans && len(this.values) == 0) {
					break loop
				}
			}
		}

		// Await children
		this.notifyScans()
		for ; n > 0; n-- {
			<-this.childChannel
		}
	})
}

func (this *IntersectScan) ChildChannel() StopChannel {
	return this.childChannel
}

func (this *IntersectScan) processKey(item value.AnnotatedValue, context *Context) bool {
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
		delete(this.values, key)
		return this.sendItem(item)
	}

	this.counts[key] = count + 1

	if count == 0 {
		this.values[key] = item
	}

	return true
}

func (this *IntersectScan) notifyScans() {
	for _, s := range this.scans {
		select {
		case s.StopChannel() <- false:
		default:
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

func (this *IntersectScan) Done() {
	for s, scan := range this.scans {
		scan.Done()
		this.scans[s] = nil
	}
	_INDEX_SCAN_POOL.Put(this.scans)
	this.scans = nil
}
