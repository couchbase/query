//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"encoding/json"

	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

// Receive subquery results
// This is a cross between Collect and Channel, where an executor gets documents on demand.
// The operator is dormant and is supposed to be at the end of a Sequence
// Its only job is to start the producer, as such it relies on the Sequence to be active
// and determine if it should executed.
type Receive struct {
	base
	plan *plan.Receive
}

func NewReceive(plan *plan.Receive, context *Context) *Receive {
	rv := &Receive{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.base.setInline()
	rv.dormant()
	rv.output = rv
	return rv
}

func (this *Receive) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitReceive(this)
}

func (this *Receive) Copy() Operator {
	rv := &Receive{
		plan: this.plan,
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *Receive) PlanOp() plan.Operator {
	return this.plan
}

func (this *Receive) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		this.fork(this.input, context, parent)
	})
}

func (this *Receive) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *Receive) reopen(context *Context) bool {
	rv := this.baseReopen(context)
	return rv
}

func (this *Receive) SendAction(action opAction) {
	this.baseSendAction(action)
	if action == _ACTION_STOP {
		this.notifyParent()
	}
}
