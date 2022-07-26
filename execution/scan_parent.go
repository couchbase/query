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

	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type ParentScan struct {
	base
	plan *plan.ParentScan
}

func NewParentScan(plan *plan.ParentScan, context *Context) *ParentScan {
	rv := &ParentScan{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *ParentScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitParentScan(this)
}

func (this *ParentScan) Copy() Operator {
	rv := &ParentScan{
		plan: this.plan,
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *ParentScan) PlanOp() plan.Operator {
	return this.plan
}

func (this *ParentScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		this.switchPhase(_EXECTIME)
		defer this.cleanup(context)
		if !active {
			return
		}

		// Shallow copy of the parent includes
		// correlated and annotated aspects
		this.sendItem(parent.Copy().(value.AnnotatedValue))
	})
}

func (this *ParentScan) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
