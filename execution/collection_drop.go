//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"encoding/json"

	//	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type DropCollection struct {
	base
	plan *plan.DropCollection
}

func NewDropCollection(plan *plan.DropCollection, context *Context) *DropCollection {
	rv := &DropCollection{
		plan: plan,
	}

	newRedirectBase(&rv.base)
	rv.output = rv
	return rv
}

func (this *DropCollection) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropCollection(this)
}

func (this *DropCollection) Copy() Operator {
	rv := &DropCollection{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *DropCollection) PlanOp() plan.Operator {
	return this.plan
}

func (this *DropCollection) RunOnce(context *Context, parent value.Value) {
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

		// Actually drop collection
		this.switchPhase(_SERVTIME)
		err := this.plan.Scope().DropCollection(this.plan.Node().Name())
		if err != nil {
			if !errors.IsCollectionNotFoundError(err) || this.plan.Node().FailIfNotExists() {
				context.Error(err)
			}
		}
	})
}

func (this *DropCollection) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
