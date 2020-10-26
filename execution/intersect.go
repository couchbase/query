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

type Intersect struct {
	base
	plan   *plan.IntersectAll
	first  Operator
	second Operator
	set    *value.Set
}

func NewIntersect(plan *plan.IntersectAll, context *Context, first, second Operator) *Intersect {
	rv := &Intersect{
		plan:   plan,
		first:  first,
		second: second,
	}

	newBase(&rv.base, context)
	rv.trackChildren(2)
	rv.output = rv
	return rv
}

func (this *Intersect) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIntersect(this)
}

func (this *Intersect) Copy() Operator {
	rv := &Intersect{
		plan:   this.plan,
		first:  this.first.Copy(),
		second: this.second.Copy(),
	}

	this.base.copy(&rv.base)
	return rv
}

func (this *Intersect) PlanOp() plan.Operator {
	return this.plan
}

func (this *Intersect) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Intersect) beforeItems(context *Context, parent value.Value) bool {
	if !context.assert(this.first != nil && this.second != nil, "Intersect has no children") {
		return false
	}

	// FIXME: should this be handled by the planner?
	distinct := NewDistinct(plan.NewDistinct(plan.PLAN_COST_NOT_AVAIL, plan.PLAN_CARD_NOT_AVAIL), context, true)
	sequence := NewSequence(plan.NewSequence(), context, this.second, distinct)
	sequence.SetParent(this)
	this.fork(sequence, context, parent)

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

func (this *Intersect) processItem(item value.AnnotatedValue, context *Context) bool {
	if !this.set.Has(item) {
		item.Recycle()
		return true
	}
	return this.sendItem(item)
}

func (this *Intersect) afterItems(context *Context) {
	this.set = nil
	context.SetSortCount(0)
}

func (this *Intersect) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
		r["first"] = this.first
		r["second"] = this.second
	})
	return json.Marshal(r)
}

func (this *Intersect) accrueTimes(o Operator) {
	if baseAccrueTimes(this, o) {
		return
	}
	copy, _ := o.(*Intersect)
	this.first.accrueTimes(copy.first)
	this.second.accrueTimes(copy.second)
}

func (this *Intersect) SendAction(action opAction) {
	rv := this.baseSendAction(action)
	first := this.first
	second := this.second
	if rv && first != nil {
		first.SendAction(action)
	}
	if rv && second != nil {
		second.SendAction(action)
	}
}

func (this *Intersect) reopen(context *Context) bool {
	rv := this.baseReopen(context)
	if rv && this.first != nil {
		rv = this.first.reopen(context)
	}
	if rv && this.second != nil {
		rv = this.second.reopen(context)
	}
	return rv
}

func (this *Intersect) Done() {
	this.baseDone()
	if this.first != nil {
		first := this.first
		this.first = nil
		first.Done()
	}
	if this.second != nil {
		second := this.second
		this.second = nil
		second.Done()
	}
}

type IntersectAll struct {
	base
	plan   *plan.IntersectAll
	first  Operator
	second Operator
	mset   *value.MultiSet
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

func (this *IntersectAll) PlanOp() plan.Operator {
	return this.plan
}

func (this *IntersectAll) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *IntersectAll) beforeItems(context *Context, parent value.Value) bool {
	if !context.assert(this.first != nil && this.second != nil, "Intersect has no children") {
		return false
	}

	// FIXME: should this be handled by the planner?
	all := NewAll(plan.NewAll(), context, true)
	sequence := NewSequence(plan.NewSequence(), context, this.second, all)
	sequence.SetParent(this)
	this.fork(sequence, context, parent)

	// we only need to wait for the first child to end
	// after that no other value will qualify anyway
	if !this.childrenWait(1) {
		this.notifyStop()
		notifyChildren(sequence)
		return false
	}

	this.mset = all.MultiSet()
	if this.mset.Len() == 0 {
		return false
	}

	this.SetInput(this.first.Output())
	this.SetStop(this.first)
	return true
}

func (this *IntersectAll) processItem(item value.AnnotatedValue, context *Context) bool {
	if !this.mset.Has(item) {
		item.Recycle()
		return true
	}
	this.mset.Remove(item.(value.Value))
	return this.sendItem(item)
}

func (this *IntersectAll) afterItems(context *Context) {
	this.mset = nil
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

func (this *IntersectAll) SendAction(action opAction) {
	rv := this.baseSendAction(action)
	first := this.first
	second := this.second
	if rv && first != nil {
		first.SendAction(action)
	}
	if rv && second != nil {
		second.SendAction(action)
	}
}

func (this *IntersectAll) reopen(context *Context) bool {
	rv := this.baseReopen(context)
	if rv && this.first != nil {
		rv = this.first.reopen(context)
	}
	if rv && this.second != nil {
		rv = this.second.reopen(context)
	}
	return rv
}

func (this *IntersectAll) Done() {
	this.baseDone()
	if this.first != nil {
		first := this.first
		this.first = nil
		first.Done()
	}
	if this.second != nil {
		second := this.second
		this.second = nil
		second.Done()
	}
}
