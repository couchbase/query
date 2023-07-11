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

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type AlterIndex struct {
	base
	plan *plan.AlterIndex
}

func NewAlterIndex(plan *plan.AlterIndex, context *Context) *AlterIndex {
	rv := &AlterIndex{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *AlterIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAlterIndex(this)
}

func (this *AlterIndex) Copy() Operator {
	rv := &AlterIndex{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *AlterIndex) PlanOp() plan.Operator {
	return this.plan
}

func (this *AlterIndex) RunOnce(context *Context, parent value.Value) {
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

		// Actually alter index
		this.switchPhase(_SERVTIME)
		node := this.plan.Node()

		index, ok := this.plan.Index().(datastore.Index3)

		// if Index does not exist
		if index == nil {
			// an error might have been generated during plan creation
			// but its reporting deferred
			defErr := this.plan.DeferredError()

			if defErr == nil {
				defErr = errors.NewCbIndexNotFoundError(this.plan.Node().Name())
			}

			context.Error(defErr)
			return
		}

		if !ok {
			context.Error(errors.NewAlterIndexError())
			return
		}

		_, err := index.Alter(context.RequestId(), node.With())
		if err != nil {
			context.Error(err)
			return
		}

	})
}

func (this *AlterIndex) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
