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

type RevokeRole struct {
	base
	plan *plan.RevokeRole
}

func NewRevokeRole(plan *plan.RevokeRole, context *Context) *RevokeRole {
	rv := &RevokeRole{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *RevokeRole) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitRevokeRole(this)
}

func (this *RevokeRole) Copy() Operator {
	rv := &RevokeRole{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *RevokeRole) PlanOp() plan.Operator {
	return this.plan
}

func (this *RevokeRole) RunOnce(context *Context, parent value.Value) {
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

		// Get the current set of users (with their role information),
		// and create a map of them by domain:userid.
		userMap, err := getUserMap(context.datastore)
		if err != nil {
			context.Fatal(err)
			return
		}

		// Create the set of deletable roles.
		roleList, err := getRoles(this.plan.Node())
		if err != nil {
			context.Fatal(err)
			return
		}

		deleteRoleMap := make(map[datastore.Role]bool, len(roleList))
		for _, role := range roleList {
			deleteRoleMap[role] = true
		}

		// Get the list of all valid roles, and verify that the roles to be
		// deleted are proper.
		validRoles, err := context.datastore.GetRolesAll()
		if err != nil {
			context.Fatal(err)
			return
		}
		err = validateRoles(roleList, validRoles)
		if err != nil {
			context.Fatal(err)
			return
		}

		// Since we only want to update each user once, even if the
		// statement mentions the user multiple times, create a map
		// of the input user ids.
		updateUserIdMap := getUsersMap(this.plan.Node().Users())

		for userId, _ := range updateUserIdMap {
			user := userMap[userId]
			if user == nil {
				context.Error(errors.NewUserNotFoundError(userId))
				continue
			}
			// Check whether this user has all the roles we are trying to delete
			// from them. Issue warning about any roles they do not have.
		eachDeleteRole:
			for deleteRole := range deleteRoleMap {
				for _, curRole := range user.Roles {
					if curRole == deleteRole {
						continue eachDeleteRole
					}
				}
				context.Warning(errors.NewRoleNotPresent(userId, deleteRole.Name, deleteRole.Target))
			}

			// Create a new list of roles for the user: their current
			// roles, minus the roles targeted for deletion.
			newRoles := make([]datastore.Role, 0, len(user.Roles))
			for _, role := range user.Roles {
				if deleteRoleMap[role] {
					continue
				}
				newRoles = append(newRoles, role)
			}
			// Issue a warning if the user now has no roles at all, an unusual and perhaps unexpected condition.
			if len(newRoles) == 0 {
				context.Warning(errors.NewUserWithNoRoles(userId))
			}
			user.Roles = newRoles
			// Update the user with their new roles on the backend.
			err = context.datastore.PutUserInfo(user)
			if err != nil {
				context.Error(err)
			}
		}

	})
}

func (this *RevokeRole) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
