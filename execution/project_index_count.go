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

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type IndexCountProject struct {
	base
	plan *plan.IndexCountProject
}

func NewIndexCountProject(plan *plan.IndexCountProject, context *Context) *IndexCountProject {
	rv := &IndexCountProject{
		plan: plan,
	}

	newBase(&rv.base, context)
	rv.execPhase = PROJECT
	rv.output = rv
	return rv
}

func (this *IndexCountProject) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexCountProject(this)
}

func (this *IndexCountProject) Copy() Operator {
	rv := &IndexCountProject{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *IndexCountProject) PlanOp() plan.Operator {
	return this.plan
}

func (this *IndexCountProject) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent, nil)
}

func (this *IndexCountProject) processItem(item value.AnnotatedValue, context *Context) bool {
	if this.plan.Projection().Raw() {
		if context.UseRequestQuota() && context.TrackValueSize(item.Size()) {
			context.Error(errors.NewMemoryQuotaExceededError())
			item.Recycle()
			return false
		}
		return this.sendItem(item)
	} else {
		var v value.Value
		var err error

		sv := value.NewScopeValue(make(map[string]interface{}, len(this.plan.Terms())), item)
		for _, term := range this.plan.Terms() {
			switch term.Result().Expression().(type) {
			case *algebra.Count:
				v = item.GetValue()
			default:
				v, err = term.Result().Expression().Evaluate(item, context)
				if err != nil {
					context.Error(errors.NewEvaluationError(err, "projection"))
					return false
				}
			}

			sv.SetField(term.Result().Alias(), v)
			if term.Result().As() != "" {
				sv.SetField(term.Result().As(), v)
			}
		}
		av := value.NewAnnotatedValue(sv)
		if context.UseRequestQuota() && context.TrackValueSize(av.Size()) {
			context.Error(errors.NewMemoryQuotaExceededError())
			av.Recycle()
			return false
		}
		return this.sendItem(av)
	}
}

func (this *IndexCountProject) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
