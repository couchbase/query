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

type DropFunction struct {
	base
	plan *plan.DropFunction
}

func NewDropFunction(plan *plan.DropFunction, context *Context) *DropFunction {
	rv := &DropFunction{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *DropFunction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropFunction(this)
}

func (this *DropFunction) Copy() Operator {
	rv := &DropFunction{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *DropFunction) PlanOp() plan.Operator {
	return this.plan
}

func (this *DropFunction) RunOnce(context *Context, parent value.Value) {
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

		// Actually drop function
		this.switchPhase(_SERVTIME)
		err := functions.DeleteFunction(this.plan.Name(), &this.operatorCtx)
		this.switchPhase(_EXECTIME)
		if err != nil {
			if this.plan.FailIfNotExists() || err.Code() != errors.E_MISSING_FUNCTION {
				context.Error(err)
			}
		}
	})
}

func (this *DropFunction) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
