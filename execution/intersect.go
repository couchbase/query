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
	"github.com/couchbase/query/value"
)

type IntersectAll struct {
	base
	first        Operator
	second       Operator
	childChannel StopChannel
	set          *value.Set
}

func NewIntersectAll(first, second Operator) *IntersectAll {
	rv := &IntersectAll{
		base:         newBase(),
		first:        first,
		second:       second,
		childChannel: make(StopChannel, 2),
	}

	rv.output = rv
	return rv
}

func (this *IntersectAll) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIntersectAll(this)
}

func (this *IntersectAll) Copy() Operator {
	rv := &IntersectAll{
		base:         this.base.copy(),
		first:        this.first.Copy(),
		second:       this.second.Copy(),
		childChannel: make(StopChannel, 2),
	}

	return rv
}

func (this *IntersectAll) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *IntersectAll) beforeItems(context *Context, parent value.Value) bool {
	distinct := NewDistinct(nil, true)
	sequence := NewSequence(this.second, distinct)
	sequence.SetParent(this)
	go sequence.RunOnce(context, parent)

	stopped := false
loop:
	for {
		select {
		case <-this.childChannel: // Never closed
			// Wait for child
			break loop
		case <-this.stopChannel: // Never closed
			stopped = true
			this.notifyStop()
			notifyChildren(sequence)
		}
	}

	if stopped {
		return false
	}

	this.set = distinct.Set()
	if this.set.Len() == 0 {
		return false
	}

	this.SetInput(this.first.Output())
	this.SetStop(this.first)
	return true
}

func (this *IntersectAll) processItem(item value.AnnotatedValue, context *Context) bool {
	return !this.set.Has(item) || this.sendItem(item)
}

func (this *IntersectAll) afterItems(context *Context) {
	this.set = nil
	context.SetSortCount(0)
}

func (this *IntersectAll) ChildChannel() StopChannel {
	return this.childChannel
}
