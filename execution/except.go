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

type Except struct {
	base
	plan   *plan.ExceptAll
	first  Operator
	second Operator
	set    *value.Set
}

func NewExcept(plan *plan.ExceptAll, context *Context, first, second Operator) *Except {
	rv := &Except{
		plan:   plan,
		first:  first,
		second: second,
	}

	newBase(&rv.base, context)
	rv.trackChildren(2)
	rv.output = rv
	return rv
}

func (this *Except) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExcept(this)
}

func (this *Except) Copy() Operator {
	rv := &Except{
		plan:   this.plan,
		first:  this.first.Copy(),
		second: this.second.Copy(),
	}

	this.base.copy(&rv.base)
	return rv
}

func (this *Except) PlanOp() plan.Operator {
	return this.plan
}

func (this *Except) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Except) beforeItems(context *Context, parent value.Value) bool {
	if !context.assert(this.first != nil && this.second != nil, "Except has no children") {
		return false
	}

	// FIXME: this should be handled by the planner
	distinct := NewDistinct(plan.NewDistinct(plan.PLAN_COST_NOT_AVAIL, plan.PLAN_CARD_NOT_AVAIL), context, true)
	sequence := NewSequence(plan.NewSequence(), context, this.second, distinct)
	sequence.SetParent(this)
	this.fork(sequence, context, parent)

	if !this.childrenWait(1) {
		this.notifyStop()
		notifyChildren(sequence)
		return false
	}

	this.set = distinct.Set()
	this.SetInput(this.first.Output())
	this.SetStop(this.first)
	return true
}

func (this *Except) processItem(item value.AnnotatedValue, context *Context) bool {
	if this.set.Has(item) {
		item.Recycle()
		return true
	}
	return this.sendItem(item)
}

func (this *Except) afterItems(context *Context) {
	this.set = nil
	context.SetSortCount(0)
}

func (this *Except) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
		r["first"] = this.first
		r["second"] = this.second
	})
	return json.Marshal(r)
}

func (this *Except) accrueTimes(o Operator) {
	if baseAccrueTimes(this, o) {
		return
	}
	copy, _ := o.(*Except)
	this.first.accrueTimes(copy.first)
	this.second.accrueTimes(copy.second)
}

func (this *Except) SendAction(action opAction) {
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

func (this *Except) reopen(context *Context) bool {
	rv := this.baseReopen(context)
	if rv && this.first != nil {
		rv = this.first.reopen(context)
	}
	if rv && this.second != nil {
		rv = this.second.reopen(context)
	}
	return rv
}

func (this *Except) Done() {
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

type ExceptAll struct {
	base
	plan   *plan.ExceptAll
	first  Operator
	second Operator
	mset   *value.MultiSet
}

func NewExceptAll(plan *plan.ExceptAll, context *Context, first, second Operator) *ExceptAll {
	rv := &ExceptAll{
		plan:   plan,
		first:  first,
		second: second,
	}

	newBase(&rv.base, context)
	rv.trackChildren(2)
	rv.output = rv
	return rv
}

func (this *ExceptAll) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExceptAll(this)
}

func (this *ExceptAll) Copy() Operator {
	rv := &ExceptAll{
		plan:   this.plan,
		first:  this.first.Copy(),
		second: this.second.Copy(),
	}

	this.base.copy(&rv.base)
	return rv
}

func (this *ExceptAll) PlanOp() plan.Operator {
	return this.plan
}

func (this *ExceptAll) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *ExceptAll) beforeItems(context *Context, parent value.Value) bool {
	if !context.assert(this.first != nil && this.second != nil, "Except has no children") {
		return false
	}

	// FIXME: this should be handled by the planner
	all := NewAll(plan.NewAll(), context, true)
	sequence := NewSequence(plan.NewSequence(), context, this.second, all)
	sequence.SetParent(this)
	this.fork(sequence, context, parent)

	if !this.childrenWait(1) {
		this.notifyStop()
		notifyChildren(sequence)
		return false
	}

	this.mset = all.MultiSet()
	this.SetInput(this.first.Output())
	this.SetStop(this.first)
	return true
}

func (this *ExceptAll) processItem(item value.AnnotatedValue, context *Context) bool {
	if this.mset.Has(item) {
		this.mset.Remove(item.(value.Value))
		item.Recycle()
		return true
	}
	return this.sendItem(item)
}

func (this *ExceptAll) afterItems(context *Context) {
	this.mset = nil
	context.SetSortCount(0)
}

func (this *ExceptAll) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
		r["first"] = this.first
		r["second"] = this.second
	})
	return json.Marshal(r)
}

func (this *ExceptAll) accrueTimes(o Operator) {
	if baseAccrueTimes(this, o) {
		return
	}
	copy, _ := o.(*ExceptAll)
	this.first.accrueTimes(copy.first)
	this.second.accrueTimes(copy.second)
}

func (this *ExceptAll) SendAction(action opAction) {
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

func (this *ExceptAll) reopen(context *Context) bool {
	rv := this.baseReopen(context)
	if rv && this.first != nil {
		rv = this.first.reopen(context)
	}
	if rv && this.second != nil {
		rv = this.second.reopen(context)
	}
	return rv
}

func (this *ExceptAll) Done() {
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
