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

type IntersectAll struct {
	base
	plan   *plan.IntersectAll
	first  Operator
	second Operator
	set    *value.Set
}

func NewIntersectAll(plan *plan.IntersectAll, context *Context, first, second Operator) *IntersectAll {
	rv := &IntersectAll{
		plan:   plan,
		first:  first,
		second: second,
	}

	newBase(&rv.base, context)
	rv.trackChildren(2)
	rv.output = rv
	return rv
}

func (this *IntersectAll) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIntersectAll(this)
}

func (this *IntersectAll) Copy() Operator {
	rv := &IntersectAll{
		plan:   this.plan,
		first:  this.first.Copy(),
		second: this.second.Copy(),
	}

	this.base.copy(&rv.base)
	return rv
}

func (this *IntersectAll) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *IntersectAll) beforeItems(context *Context, parent value.Value) bool {
	if !context.assert(this.first != nil && this.second != nil, "Intersect has no children") {
		return false
	}

	// FIXME: should this be handled by the planner?
	distinct := NewDistinct(plan.NewDistinct(), context, true)
	sequence := NewSequence(plan.NewSequence(), context, this.second, distinct)
	sequence.SetParent(this)
	go sequence.RunOnce(context, parent)

	// we only need to wait for the first child to end
	// after that no other value will qualify anyway
	if !this.childrenWait(1) {
		this.notifyStop()
		notifyChildren(sequence)
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
	if !this.set.Has(item) {
		item.Recycle()
		return true
	}
	return this.sendItem(item)
}

func (this *IntersectAll) afterItems(context *Context) {
	this.set = nil
	context.SetSortCount(0)
}

func (this *IntersectAll) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
		r["first"] = this.first
		r["second"] = this.second
	})
	return json.Marshal(r)
}

func (this *IntersectAll) accrueTimes(o Operator) {
	if baseAccrueTimes(this, o) {
		return
	}
	copy, _ := o.(*IntersectAll)
	this.first.accrueTimes(copy.first)
	this.second.accrueTimes(copy.second)
}

func (this *IntersectAll) SendStop() {
	this.baseSendStop()
	first := this.first
	second := this.second
	if first != nil {
		first.SendStop()
	}
	if second != nil {
		second.SendStop()
	}
}

func (this *IntersectAll) reopen(context *Context) {
	this.baseReopen(context)
	if this.first != nil {
		this.first.reopen(context)
	}
	if this.second != nil {
		this.second.reopen(context)
	}
}

func (this *IntersectAll) Done() {
	this.baseDone()
	if this.first != nil {
		this.first.Done()
	}
	if this.second != nil {
		this.second.Done()
	}
	this.first = nil
	this.second = nil
}
