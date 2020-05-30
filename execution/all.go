//  Copyright (c) 2019 Couchbase, Inc.
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

type All struct {
	base
	mset    *value.MultiSet
	plan    *plan.All
	collect bool
}

func NewAll(plan *plan.All, context *Context, collect bool) *All {
	rv := &All{
		mset:    value.NewMultiSet(int(context.GetPipelineCap()), false, false),
		plan:    plan,
		collect: collect,
	}

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *All) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAll(this)
}

func (this *All) Copy() Operator {
	cap := int(GetPipelineCap())
	if this.mset != nil {
		cap = this.mset.ObjectCap()
	}
	rv := &All{
		plan: this.plan,
		mset: value.NewMultiSet(cap, this.collect, false),
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *All) PlanOp() plan.Operator {
	return this.plan
}

func (this *All) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *All) processItem(item value.AnnotatedValue, context *Context) bool {
	p := item.GetAttachment("projection")
	if p == nil {
		p = item
	}

	this.mset.Add(p.(value.Value))
	return true
}

func (this *All) afterItems(context *Context) {
	if !this.collect {
		this.mset = nil
	}
}

func (this *All) MultiSet() *value.MultiSet {
	return this.mset
}

func (this *All) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *All) reopen(context *Context) bool {
	rv := this.baseReopen(context)
	this.mset = value.NewMultiSet(int(context.GetPipelineCap()), false, false)
	return rv
}
