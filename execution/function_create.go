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

type CreateFunction struct {
	base
	plan *plan.CreateFunction
}

func NewCreateFunction(plan *plan.CreateFunction, context *Context) *CreateFunction {
	rv := &CreateFunction{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *CreateFunction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateFunction(this)
}

func (this *CreateFunction) Copy() Operator {
	rv := &CreateFunction{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *CreateFunction) PlanOp() plan.Operator {
	return this.plan
}

func (this *CreateFunction) RunOnce(context *Context, parent value.Value) {
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

		// Actually create function
		var err errors.Error

		// If the function is an Internal JS function, load the JS function body, to check if it is syntactically correct
		err = this.plan.Body().Load(this.plan.Name())

		if err != nil {
			this.plan.Body().Unload(this.plan.Name())
			context.Error(err)
			return
		}

		replace := this.plan.Replace()
		if !this.plan.FailIfExists() {
			// IF EXISTS clause has been specified so ensure replacement isn't attempted
			replace = false
		}
		if replace {
			err = functions.CheckDelete(this.plan.Name(), context)
		}
		if err == nil {
			err = this.plan.Body().SetStorage(context, this.plan.Name().Path())
		}
		if err == nil {
			this.switchPhase(_SERVTIME)
			err = functions.AddFunction(this.plan.Name(), this.plan.Body(), replace)
			this.switchPhase(_EXECTIME)
		}
		if err != nil && (this.plan.FailIfExists() || err.Code() != errors.E_DUPLICATE_FUNCTION) {
			context.Error(err)
		}
	})
}

func (this *CreateFunction) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
