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

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type Explain struct {
	base
	plan plan.Operator
}

func NewExplain(plan plan.Operator, context *Context) *Explain {
	rv := &Explain{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *Explain) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExplain(this)
}

func (this *Explain) Copy() Operator {
	rv := &Explain{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *Explain) PlanOp() plan.Operator {
	return this.plan
}

func (this *Explain) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		defer this.notify() // Notify that I have stopped
		if !active {
			return
		}

		bytes, err := this.plan.MarshalJSON()
		if err != nil {
			context.Fatal(errors.NewExplainError(err, "EXPLAIN: Error marshaling JSON."))
			return
		}

		value := value.NewAnnotatedValue(bytes)
		this.sendItem(value)

	})
}

func (this *Explain) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *Explain) Done() {
	this.baseDone()
	this.plan = nil
}
