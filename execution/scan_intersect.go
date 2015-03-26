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
	"github.com/couchbase/query/value"
)

type IntersectScan struct {
	base
	scans        []Operator
	counts       map[string]int
	values       map[string]value.AnnotatedValue
	childChannel StopChannel
}

func NewIntersectScan(scans []Operator) *IntersectScan {
	rv := &IntersectScan{
		base:         newBase(),
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
	scans := make([]Operator, len(this.scans))
	for i, s := range this.scans {
		scans[i] = s.Copy()
	}

	return &IntersectScan{
		base:         this.base.copy(),
		scans:        scans,
		childChannel: make(StopChannel, len(scans)),
	}
}

func (this *IntersectScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover()       // Recover from any panic
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped
		defer func() { this.counts = nil }()
		defer func() { this.values = nil }()

		this.counts = make(map[string]int, 1024)
		this.values = make(map[string]value.AnnotatedValue, 1024)

		channel := NewChannel()

		for _, scan := range this.scans {
			scan.SetParent(this)
			scan.SetOutput(channel)
			go scan.RunOnce(context, parent)
		}

		var item value.AnnotatedValue
		n := len(this.scans)
		stopped := false
		ok := true
	loop:
		for ok {
			select {
			case <-this.stopChannel:
				stopped = true
				break loop
			default:
			}

			select {
			case item, ok = <-channel.ItemChannel():
				if ok {
					ok = this.processKey(item, context)
				}
			case <-this.childChannel:
				if n == len(this.scans) {
					this.notifyScans()
				}
				n--
			case <-this.stopChannel:
				stopped = true
				break loop
			default:
				if n < len(this.scans) {
					break loop
				}
			}
		}

		this.values = nil
		if n == len(this.scans) {
			this.notifyScans()
		}

		// Await children
		for ; n > 0; n-- {
			<-this.childChannel
		}

		select {
		case channel.StopChannel() <- false:
		default:
		}

		if !stopped {
			this.sendItems()
		}

		this.values = nil
		this.counts = nil
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

func (this *IntersectScan) sendItems() {
	for _, av := range this.values {
		if !this.sendItem(av) {
			return
		}
	}
}

func (this *IntersectScan) notifyScans() {
	for _, s := range this.scans {
		select {
		case s.StopChannel() <- false:
		default:
		}
	}
}
