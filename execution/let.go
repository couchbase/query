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

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type Let struct {
	base
	plan *plan.Let
}

func NewLet(plan *plan.Let, context *Context) *Let {
	rv := &Let{
		plan: plan,
	}

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *Let) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitLet(this)
}

func (this *Let) Copy() Operator {
	rv := &Let{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *Let) PlanOp() plan.Operator {
	return this.plan
}

func (this *Let) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Let) processItem(item value.AnnotatedValue, context *Context) bool {
	lv := item.Copy().(value.AnnotatedValue)
	for _, b := range this.plan.Bindings() {
		v, e := b.Expression().Evaluate(lv, context)
		if e != nil {
			context.Error(errors.NewEvaluationError(e, "LET"))
			return false
		}

		lv.SetField(b.Variable(), v)
	}

	item.Recycle()
	return this.sendItem(lv)
}

func (this *Let) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
