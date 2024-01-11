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

type AlterUser struct {
	base
	plan *plan.AlterUser
}

func NewAlterUser(plan *plan.AlterUser, context *Context) *AlterUser {
	rv := &AlterUser{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *AlterUser) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAlterUser(this)
}

func (this *AlterUser) Copy() Operator {
	rv := &AlterUser{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *AlterUser) PlanOp() plan.Operator {
	return this.plan
}

func (this *AlterUser) RunOnce(context *Context, parent value.Value) {
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
			context.Error(errors.NewUserNotFoundError(u.Domain + ":" + u.Id))
		} else {
			p, ok := this.plan.Node().Password()
			if ok {
				v, err := p.Evaluate(parent, &this.operatorCtx)
				if err != nil {
					context.Error(errors.NewEvaluationError(err, "password"))
					return
				}
				u.Password = v.ToString()
			} else {
				u.Password = string([]byte{0})
			}
			if g, ok := this.plan.Node().Groups(); ok {
				u.Groups = g
			}
			if n, ok := this.plan.Node().Name(); ok {
				u.Name = n
			} else {
				u.Name = string([]byte{0})
			}

			err := context.datastore.PutUserInfo(&u)
			if err != nil {
				context.Error(err)
			}
		}
	})
}

func (this *AlterUser) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
