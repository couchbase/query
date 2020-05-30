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

// Distincting of input data.
type Distinct struct {
	base
	set     *value.Set
	plan    *plan.Distinct
	collect bool
}

func NewDistinct(plan *plan.Distinct, context *Context, collect bool) *Distinct {
	rv := &Distinct{
		set:     value.NewSet(int(context.GetPipelineCap()), false, false),
		plan:    plan,
		collect: collect,
	}

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *Distinct) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDistinct(this)
}

func (this *Distinct) Copy() Operator {
	cap := int(GetPipelineCap())
	if this.set != nil {
		cap = this.set.ObjectCap()
	}
	rv := &Distinct{
		plan: this.plan,
		set:  value.NewSet(cap, false, false),
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *Distinct) PlanOp() plan.Operator {
	return this.plan
}

func (this *Distinct) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Distinct) processItem(item value.AnnotatedValue, context *Context) bool {
	p := item.GetAttachment("projection")
	if p == nil {
		p = item
	}

	if !this.set.Has(p.(value.Value)) {
		this.set.Put(p.(value.Value), item)
		return this.collect || this.sendItem(item)
	} else {
		item.Recycle()
	}
	return true
}

func (this *Distinct) afterItems(context *Context) {
	if !this.collect {
		this.set = nil
	}
}

func (this *Distinct) Set() *value.Set {
	return this.set
}

func (this *Distinct) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *Distinct) reopen(context *Context) bool {
	rv := this.baseReopen(context)
	this.set = value.NewSet(int(context.GetPipelineCap()), false, false)
	return rv
}
