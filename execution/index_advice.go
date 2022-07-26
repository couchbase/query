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
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type IndexAdvice struct {
	base
	plan plan.Operator
}

func NewIndexAdvisor(plan plan.Operator, context *Context) *IndexAdvice {
	rv := &IndexAdvice{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *IndexAdvice) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexAdvice(this)
}

func (this *IndexAdvice) Copy() Operator {
	rv := &IndexAdvice{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *IndexAdvice) PlanOp() plan.Operator {
	return this.plan
}

func (this *IndexAdvice) RunOnce(context *Context, parent value.Value) {
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

		value := value.NewAnnotatedValue(parent)
		this.sendItem(value)

	})
}

func (this *IndexAdvice) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *IndexAdvice) Done() {
	this.baseDone()
	this.plan = nil
}
