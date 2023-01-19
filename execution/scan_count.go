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

type CountScan struct {
	base
	plan *plan.CountScan
}

func NewCountScan(plan *plan.CountScan, context *Context) *CountScan {
	rv := &CountScan{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *CountScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCountScan(this)
}

func (this *CountScan) Copy() Operator {
	rv := &CountScan{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *CountScan) PlanOp() plan.Operator {
	return this.plan
}

func (this *CountScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		this.switchPhase(_EXECTIME)
		this.setExecPhase(COUNT, context)
		defer this.cleanup(context)
		if !active {
			return
		}

		this.switchPhase(_SERVTIME)
		count, e := this.plan.Keyspace().Count(context)
		this.switchPhase(_EXECTIME)

		if e != nil {
			context.Error(e)
			return
		}

		cv := value.NewScopeValue(nil, parent)
		av := value.NewAnnotatedValue(cv)
		av.SetAttachment("count", value.NewValue(count))
		if context.UseRequestQuota() {
			err := context.TrackValueSize(av.Size())
			if err != nil {
				context.Error(err)
				av.Recycle()
				return
			}
		}
		this.sendItem(av)
	})
}

func (this *CountScan) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
