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
	"fmt"
	"math"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type Limit struct {
	base
	plan  *plan.Limit
	limit int64
}

func NewLimit(plan *plan.Limit, context *Context) *Limit {
	rv := &Limit{
		plan: plan,
	}

	// MB-27945 limit does not run inside a parallel group
	// serialize only if parallelism is off
	if context.MaxParallelism() == 1 {
		newSerializedBase(&rv.base, context)
	} else {
		newBase(&rv.base, context)
	}
	rv.output = rv
	return rv
}

func (this *Limit) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitLimit(this)
}

func (this *Limit) Copy() Operator {
	rv := &Limit{plan: this.plan, limit: this.limit}
	this.base.copy(&rv.base)
	return rv
}

func (this *Limit) PlanOp() plan.Operator {
	return this.plan
}

func (this *Limit) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent, nil)
}

func (this *Limit) beforeItems(context *Context, parent value.Value) bool {
	val, e := this.plan.Expression().Evaluate(parent, &this.operatorCtx)
	if e != nil {
		context.Error(errors.NewEvaluationError(e, "LIMIT"))
		return false
	}

	actual := val.Actual()
	switch actual := actual.(type) {
	case float64:
		if math.Trunc(actual) == actual {
			this.limit = int64(actual)
			return true
		}
	}

	context.Error(errors.NewInvalidValueError(
		fmt.Sprintf("Invalid LIMIT value %v.", actual)))
	return false
}

func (this *Limit) processItem(item value.AnnotatedValue, context *Context) bool {
	if this.limit > 0 {
		this.limit--
		return this.sendItem(item)
	} else {
		if context.UseRequestQuota() {
			context.ReleaseValueSize(item.Size())
		}
		item.Recycle()
		return false
	}
}

func (this *Limit) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
