//  Copyright 2023-Present Couchbase, Inc.
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
	"github.com/couchbase/query/sequences"
	"github.com/couchbase/query/value"
)

type DropSequence struct {
	base
	plan *plan.DropSequence
}

func NewDropSequence(plan *plan.DropSequence, context *Context) *DropSequence {
	rv := &DropSequence{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *DropSequence) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropSequence(this)
}

func (this *DropSequence) Copy() Operator {
	rv := &DropSequence{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *DropSequence) PlanOp() plan.Operator {
	return this.plan
}

func (this *DropSequence) RunOnce(context *Context, parent value.Value) {
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

		this.switchPhase(_SERVTIME)
		err := sequences.DropSequence(this.plan.Node().Name())
		if err != nil {
			if !errors.IsSequenceNotFoundError(err) || this.plan.Node().FailIfNotExists() {
				if err.IsWarning() {
					context.Error(err)
				} else {
					context.Error(err)
				}
			}
		}
	})
}

func (this *DropSequence) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
