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

type CreateUser struct {
	base
	plan *plan.CreateUser
}

func NewCreateUser(plan *plan.CreateUser, context *Context) *CreateUser {
	rv := &CreateUser{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *CreateUser) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateUser(this)
}

func (this *CreateUser) Copy() Operator {
	rv := &CreateUser{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *CreateUser) PlanOp() plan.Operator {
	return this.plan
}

func (this *CreateUser) RunOnce(context *Context, parent value.Value) {
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
		if err == nil {
			context.Error(errors.NewUserExistsError(u.Domain + ":" + u.Id))
		} else {
			p, ok := this.plan.Node().Password()
			if ok {
				u.Password = p
			} else {
				u.Password = string([]byte{0})
			}
			if !ok && u.Domain == "local" {
				context.Error(errors.NewUserAttributeError(u.Domain, "password", "required"))
			} else if ok && u.Domain == "external" {
				context.Error(errors.NewUserAttributeError(u.Domain, "password", "not supported"))
			} else {
				if g, ok := this.plan.Node().Groups(); ok {
					u.Groups = g
				}
				if n, ok := this.plan.Node().Name(); ok {
					u.Name = n
				} else {
					u.Name = string([]byte{0})
				}

				err = context.datastore.PutUserInfo(&u)
				if err != nil {
					context.Error(err)
				}
			}
		}
	})
}

func (this *CreateUser) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
