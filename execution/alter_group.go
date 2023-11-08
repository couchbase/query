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

type AlterGroup struct {
	base
	plan *plan.AlterGroup
}

func NewAlterGroup(plan *plan.AlterGroup, context *Context) *AlterGroup {
	rv := &AlterGroup{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *AlterGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAlterGroup(this)
}

func (this *AlterGroup) Copy() Operator {
	rv := &AlterGroup{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *AlterGroup) PlanOp() plan.Operator {
	return this.plan
}

func (this *AlterGroup) RunOnce(context *Context, parent value.Value) {
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
			if r, ok := this.plan.Node().Roles(); ok {
				g.Roles = make([]datastore.Role, len(r))
				for i := range r {
					p1 := strings.Split(r[i], "[")
					g.Roles[i].Name = p1[0]
					if len(p1) > 1 {
						g.Roles[i].Target = strings.TrimSuffix(p1[1], "]")
					}
				}
			}
			if g.Roles == nil {
				// default to empty roles list if we didn't get one back in the information and haven't specified a list
				g.Roles = make([]datastore.Role, 0, 1)
			}
			if d, ok := this.plan.Node().Desc(); ok {
				g.Desc = d
			} else {
				g.Desc = string([]byte{0})
			}

			err = context.datastore.PutGroupInfo(&g)
			if err != nil {
				context.Error(err)
			}
		}
	})
}

func (this *AlterGroup) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
