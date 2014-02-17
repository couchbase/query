//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execute

import (
	"fmt"

	"github.com/couchbaselabs/query/err"
	"github.com/couchbaselabs/query/value"
)

type MultipleScan struct {
	base
	scans        []Operator
	counts       map[string]int
	values       map[string]value.AnnotatedValue
	childChannel StopChannel
}

func NewMultipleScan(scans []Operator) *MultipleScan {
	rv := &MultipleScan{
		base:         newBase(),
		scans:        scans,
		childChannel: make(StopChannel, len(scans)),
	}

	rv.output = rv
	return rv
}

func (this *MultipleScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitMultipleScan(this)
}

func (this *MultipleScan) Copy() Operator {
	scans := make([]Operator, len(this.scans))
	for i, s := range this.scans {
		scans[i] = s.Copy()
	}

	return &MultipleScan{
		base:         this.base.copy(),
		scans:        scans,
		childChannel: make(StopChannel, len(scans)),
	}
}

func (this *MultipleScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped
		defer func() { this.counts = nil }()
		defer func() { this.values = nil }()

		this.counts = make(map[string]int, 1024)
		channel := NewChannel()

		for _, s := range this.scans {
			s.SetParent(this)
			s.SetOutput(channel)
			go s.RunOnce(context, parent)
		}

		var item value.AnnotatedValue
		n := len(this.scans)
		ok := true

		for ok {
			select {
			case item, ok = <-channel.ItemChannel():
				if ok {
					ok = this.processKey(item, context)
					if !ok {
						this.values = nil
					}
				}
			case <-this.childChannel:
				n--
				break
			case <-this.stopChannel:
				this.values = nil
				break
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

		this.sendItems()
	})
}

func (this *MultipleScan) ChildChannel() StopChannel {
	return this.childChannel
}

func (this *MultipleScan) processKey(item value.AnnotatedValue, context *Context) bool {
	m := item.GetAttachment("meta")
	meta, ok := m.(map[string]interface{})
	if !ok {
		context.ErrorChannel() <- err.NewError(nil,
			fmt.Sprintf("Missing or invalid meta %v of type %T.", m, m))
		return false
	}

	k := meta["id"]
	key, ok := k.(string)
	if !ok {
		context.ErrorChannel() <- err.NewError(nil,
			fmt.Sprintf("Missing or invalid primary key %v of type %T.", k, k))
		return false
	}

	count := this.counts[key]
	this.counts[key] = count + 1

	if count+1 == len(this.scans) {
		delete(this.values, key)
		return this.sendItem(item)
	}

	if count == 0 {
		this.values[key] = item
	}

	return true
}

func (this *MultipleScan) sendItems() {
	n := len(this.scans)
	for key, av := range this.values {
		if this.counts[key] == n && !this.sendItem(av) {
			return
		}
	}
}

func (this *MultipleScan) notifyScans() {
	for _, s := range this.scans {
		select {
		case s.StopChannel() <- false:
		default:
		}
	}
}
