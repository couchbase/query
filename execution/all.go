//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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
	this.runConsumer(this, context, parent, nil)
}

func (this *All) processItem(item value.AnnotatedValue, context *Context) bool {
	p := item.GetAttachment(value.ATT_PROJECTION)
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
