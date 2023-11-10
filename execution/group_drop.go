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

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type DropGroup struct {
	base
	plan *plan.DropGroup
}

func NewDropGroup(plan *plan.DropGroup, context *Context) *DropGroup {
	rv := &DropGroup{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *DropGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropGroup(this)
}

func (this *DropGroup) Copy() Operator {
	rv := &DropGroup{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *DropGroup) PlanOp() plan.Operator {
	return this.plan
}

func (this *DropGroup) RunOnce(context *Context, parent value.Value) {
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

		var g datastore.Group
		g.Id = this.plan.Node().Group()

		err := context.datastore.GetGroupInfo(&g)
		if err != nil {
			context.Error(errors.NewGroupNotFoundError(g.Id))
		} else {
			err := context.datastore.DeleteGroup(&g)
			if err != nil {
				context.Error(err)
			}
		}
	})
}

func (this *DropGroup) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
