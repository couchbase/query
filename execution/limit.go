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
	"github.com/couchbase/query/expression"
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
	lim, err := getLimit(this.plan.Expression(), parent, &this.operatorCtx)
	if err != nil {
		context.Error(err)
		return false
	}

	this.limit = lim
	return true
}

func (this *Limit) processItem(item value.AnnotatedValue, context *Context) bool {
	if this.limit > 0 {
		this.limit--
		return this.sendItem(item)
	} else {

		// MB-53235 for serialized operators item management rests with the producer
		if !this.serialized {
			if context.UseRequestQuota() {
				context.ReleaseValueSize(item.Size())
			}
			item.Recycle()
		}
		return false
	}
}

func (this *Limit) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func getLimit(limit expression.Expression, parent value.Value, context *opContext) (int64, errors.Error) {
	if limit == nil {
		return -1, nil
	}

	val, err := limit.Evaluate(parent, context)
	if err != nil {
		return -1, errors.NewEvaluationError(err, "LIMIT clause")
	}

	l := val.ActualForIndex() // Exact number
	switch l := l.(type) {
	case int64:
		return l, nil
	case float64:
		if math.Trunc(l) == l {
			return int64(l), nil
		}
	}

	return -1, errors.NewInvalidValueError(fmt.Sprintf("Invalid LIMIT %v of type %T.", l, l))
}
