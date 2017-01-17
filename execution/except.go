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

type ExceptAll struct {
	base
	plan         *plan.ExceptAll
	first        Operator
	second       Operator
	childChannel StopChannel
	set          *value.Set
}

func NewExceptAll(plan *plan.ExceptAll, first, second Operator) *ExceptAll {
	rv := &ExceptAll{
		base:         newBase(),
		plan:         plan,
		first:        first,
		second:       second,
		childChannel: make(StopChannel, 2),
	}

	rv.output = rv
	return rv
}

func (this *ExceptAll) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExceptAll(this)
}

func (this *ExceptAll) Copy() Operator {
	rv := &ExceptAll{
		base:         this.base.copy(),
		plan:         this.plan,
		first:        this.first.Copy(),
		second:       this.second.Copy(),
		childChannel: make(StopChannel, 2),
	}

	return rv
}

func (this *ExceptAll) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *ExceptAll) beforeItems(context *Context, parent value.Value) bool {

	// FIXME: this should be handled by the planner
	distinct := NewDistinct(plan.NewDistinct(), true)
	sequence := NewSequence(plan.NewSequence(), this.second, distinct)
	sequence.SetParent(this)
	go sequence.RunOnce(context, parent)

	stopped := false
	this.switchPhase(_CHANTIME)
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
	this.switchPhase(_EXECTIME)

	if stopped {
		return false
	}

	this.set = distinct.Set()
	this.SetInput(this.first.Output())
	this.SetStop(this.first)
	return true
}

func (this *ExceptAll) processItem(item value.AnnotatedValue, context *Context) bool {
	return this.set.Has(item) || this.sendItem(item)
}

func (this *ExceptAll) afterItems(context *Context) {
	this.set = nil
	context.SetSortCount(0)
}

func (this *ExceptAll) ChildChannel() StopChannel {
	return this.childChannel
}

func (this *ExceptAll) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
		r["first"] = this.first
		r["second"] = this.second
	})
	return json.Marshal(r)
}

func (this *ExceptAll) Done() {
	this.first.Done()
	this.second.Done()
	this.first = nil
	this.second = nil
}
