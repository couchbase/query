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
	"strings"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type DropUser struct {
	base
	plan *plan.DropUser
}

func NewDropUser(plan *plan.DropUser, context *Context) *DropUser {
	rv := &DropUser{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *DropUser) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropUser(this)
}

func (this *DropUser) Copy() Operator {
	rv := &DropUser{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *DropUser) PlanOp() plan.Operator {
	return this.plan
}

func (this *DropUser) RunOnce(context *Context, parent value.Value) {
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

		var u datastore.User
		parts := strings.Split(this.plan.Node().User(), ":")
		if len(parts) == 2 {
			u.Domain = parts[0]
			u.Id = parts[1]
		} else {
			u.Domain = "local"
			u.Id = parts[0]
		}

		err := context.datastore.GetUserInfo(&u)
		if err != nil {
			if this.plan.Node().FailIfNotExists() {
				context.Error(errors.NewUserNotFoundError(u.Domain + ":" + u.Id))
			}
		} else {
			err := context.datastore.DeleteUser(&u)
			if err != nil {
				context.Error(err)
			}
		}
	})
}

func (this *DropUser) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
