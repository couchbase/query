//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

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

	newRedirectBase(&rv.base)
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

		if this.plan.Replace() {
			err = functions.CheckDelete(this.plan.Name(), context)
		}
		if err == nil {
			this.switchPhase(_SERVTIME)
			err = functions.AddFunction(this.plan.Name(), this.plan.Body(), this.plan.Replace())
			this.switchPhase(_EXECTIME)
		}
		if err != nil {
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
