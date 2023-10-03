//  Copyright 2014-Present Couchbase, Inc.
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

// Collect subquery results
type Collect struct {
	base
	plan   *plan.Collect
	values []interface{}
}

const _COLLECT_CAP = 64

func NewCollect(plan *plan.Collect, context *Context) *Collect {
	rv := &Collect{
		plan:   plan,
		values: make([]interface{}, 0, _COLLECT_CAP),
	}

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *Collect) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCollect(this)
}

func (this *Collect) Copy() Operator {
	rv := &Collect{
		plan:   this.plan,
		values: make([]interface{}, 0, _COLLECT_CAP),
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *Collect) PlanOp() plan.Operator {
	return this.plan
}

func (this *Collect) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent, nil)
}

func (this *Collect) processItem(item value.AnnotatedValue, context *Context) bool {
	if len(this.values) == cap(this.values) {
		values := make([]interface{}, len(this.values), len(this.values)<<1)
		copy(values, this.values)
		this.values = values
	}

	this.values = append(this.values, item.Actual())
	return true
}

func (this *Collect) ValuesOnce() value.Value {
	defer this.releaseValues()
	return value.NewValue(this.values)
}

func (this *Collect) releaseValues() {
	this.values = nil
}

func (this *Collect) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *Collect) reopen(context *Context) bool {
	rv := this.baseReopen(context)
	this.values = make([]interface{}, 0, _COLLECT_CAP)
	return rv
}
