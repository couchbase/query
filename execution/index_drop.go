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

type DropIndex struct {
	base
	plan *plan.DropIndex
}

func NewDropIndex(plan *plan.DropIndex, context *Context) *DropIndex {
	rv := &DropIndex{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *DropIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropIndex(this)
}

func (this *DropIndex) Copy() Operator {
	rv := &DropIndex{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *DropIndex) PlanOp() plan.Operator {
	return this.plan
}

func (this *DropIndex) RunOnce(context *Context, parent value.Value) {
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

		// Actually drop index
		this.switchPhase(_SERVTIME)
		index := this.plan.Index()
		if index == nil {
			if this.plan.Node().FailIfNotExists() {
				err := this.plan.DeferredError()
				if err == nil {
					err = errors.NewCbIndexNotFoundError(this.plan.Node().Name())
				}
				context.Error(err)
			}
			return
		}
		if this.plan.Node() != nil && this.plan.Node().PrimaryOnly() && !index.IsPrimary() {
			context.Error(errors.NewCbNotPrimaryIndexError(this.plan.Node().Name()))
			return
		}
		err := index.Drop(context.RequestId())
		if err != nil {
			context.Error(err)
		}
	})
}

func (this *DropIndex) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
