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

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type ExecuteFunction struct {
	base
	plan *plan.ExecuteFunction
}

func NewExecuteFunction(plan *plan.ExecuteFunction, context *Context) *ExecuteFunction {
	rv := &ExecuteFunction{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *ExecuteFunction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExecuteFunction(this)
}

func (this *ExecuteFunction) Copy() Operator {
	rv := &ExecuteFunction{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *ExecuteFunction) PlanOp() plan.Operator {
	return this.plan
}

func (this *ExecuteFunction) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		defer this.notify() // Notify that I have stopped

		if !active || context.Readonly() {
			return
		}

		// evaluate the parameter list
		var args []value.Value

		exprs := this.plan.Expressions()
		l := len(exprs)
		if l > 0 {
			args = make([]value.Value, l)
			for e, _ := range exprs {
				ev, err := exprs[e].Evaluate(parent, &this.operatorCtx)
				if err != nil {
					context.Error(errors.NewEvaluationError(err, "ExecuteFunction"))
					return
				}
				args[e] = ev
			}
		}

		context.SetPreserveProjectionOrder(false)
		val, err := functions.ExecuteFunction(this.plan.Name(), functions.NONE, args, &this.operatorCtx)
		if err != nil {
			context.Error(err)
		} else {
			av := value.NewAnnotatedValue(val)
			if context.UseRequestQuota() {
				err := context.TrackValueSize(av.Size())
				if err != nil {
					context.Error(err)
					av.Recycle()
					return
				}
			}
			this.sendItem(av)
		}
	})
}

func (this *ExecuteFunction) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
