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

	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/value"
)

type UnionScan struct {
	base
	scans        []Operator
	values       map[string]value.AnnotatedValue
	childChannel StopChannel
}

func NewUnionScan(scans []Operator) *UnionScan {
	rv := &UnionScan{
		base:         newBase(),
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
	scans := make([]Operator, len(this.scans))
	for i, s := range this.scans {
		scans[i] = s.Copy()
	}

	return &UnionScan{
		base:         this.base.copy(),
		scans:        scans,
		childChannel: make(StopChannel, len(scans)),
	}
}

func (this *UnionScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover()       // Recover from any panic
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped
		defer func() { this.values = nil }()

		this.values = make(map[string]value.AnnotatedValue, 1024)

		channel := NewChannel()

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
				this.values = nil
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
				break loop
			case <-this.stopChannel:
				this.values = nil
				break loop
			}
		}

		this.notifyScans()

		// Await children
		for ; n > 0; n-- {
			<-this.childChannel
		}

		select {
		case channel.StopChannel() <- false:
		default:
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
		context.Error(errors.NewError(nil,
			fmt.Sprintf("Missing or invalid meta %v of type %T.", m, m)))
		return false
	}

	k := meta["id"]
	key, ok := k.(string)
	if !ok {
		context.Error(errors.NewError(nil,
			fmt.Sprintf("Missing or invalid primary key %v of type %T.", k, k)))
		return false
	}

	if this.values[key] != nil {
		return true
	}

	this.values[key] = item
	return this.sendItem(item)
}

func (this *UnionScan) notifyScans() {
	for _, s := range this.scans {
		select {
		case s.StopChannel() <- false:
		default:
		}
	}
}
