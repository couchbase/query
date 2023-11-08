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
	_ "fmt"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

// ValueScan is used when there is a VALUES clause, e.g. in INSERTs.
type ValueScan struct {
	base
	plan *plan.ValueScan
}

func NewValueScan(plan *plan.ValueScan, context *Context) *ValueScan {
	rv := &ValueScan{
		plan: plan,
	}

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *ValueScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitValueScan(this)
}

func (this *ValueScan) Copy() Operator {
	rv := &ValueScan{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *ValueScan) PlanOp() plan.Operator {
	return this.plan
}

func (this *ValueScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		this.switchPhase(_EXECTIME)
		defer this.cleanup(context)
		if !active {
			return
		}

		pairs := this.plan.Values()

		for _, pair := range pairs {
			key, err := pair.Key().Evaluate(parent, &this.operatorCtx)
			if err != nil {
				context.Error(errors.NewEvaluationError(err, "KEY"))
				return
			}

			val, err := pair.Value().Evaluate(parent, &this.operatorCtx)
			if err != nil {
				context.Error(errors.NewEvaluationError(err, "VALUES"))
				return
			}

			av := value.NewAnnotatedValue(nil)
			av.SetAttachment("key", key)
			av.SetAttachment("value", val)

			if pair.Options() != nil {
				options, err := pair.Options().Evaluate(parent, &this.operatorCtx)
				if err != nil {
					context.Error(errors.NewEvaluationError(err, "OPTIONS"))
					return
				}
				av.SetAttachment("options", options)
			}

			if !this.sendItem(av) {
				return
			}
		}
	})
}

func (this *ValueScan) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
