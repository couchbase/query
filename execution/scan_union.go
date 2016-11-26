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

type UnionScan struct {
	base
	plan         *plan.UnionScan
	scans        []Operator
	keys         map[string]bool
	childChannel StopChannel
}

func NewUnionScan(plan *plan.UnionScan, scans []Operator) *UnionScan {
	rv := &UnionScan{
		base:         newBase(),
		plan:         plan,
		scans:        scans,
		childChannel: make(StopChannel, len(scans)),
	}

	rv.output = rv
	return rv
}

func (this *UnionScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnionScan(this)
}

func (this *UnionScan) Copy() Operator {
	// FIXME reinstate _INDEX_SCAN_POOL if possible
	scans := make([]Operator, 0, len(this.scans))

	for i, s := range this.scans {
		scans[i] = s.Copy()
	}

	return &UnionScan{
		base:         this.base.copy(),
		plan:         this.plan,
		scans:        scans,
		childChannel: make(StopChannel, len(scans)),
	}
}

func (this *UnionScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover()       // Recover from any panic
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		this.keys = _STRING_BOOL_POOL.Get()
		defer func() {
			_STRING_BOOL_POOL.Put(this.keys)
			this.keys = nil
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
				if n == 0 {
					break loop
				}
			}
		}

		notifyChildren(this.scans...)

		// Await children
		for ; n > 0; n-- {
			<-this.childChannel
		}
	})
}

func (this *UnionScan) ChildChannel() StopChannel {
	return this.childChannel
}

func (this *UnionScan) processKey(item value.AnnotatedValue, context *Context) bool {
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

	if _, ok = this.keys[key]; ok {
		return true
	}

	this.keys[key] = true
	return this.sendItem(item)
}

func (this *UnionScan) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
		r["scans"] = this.scans
	})
	return json.Marshal(r)
}
