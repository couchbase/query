//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
		base: newRedirectBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *RevokeRole) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitRevokeRole(this)
}

func (this *RevokeRole) Copy() Operator {
	return &RevokeRole{this.base.copy(), this.plan}
}

func (this *RevokeRole) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover() // Recover from any panic
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		if context.Readonly() {
			return
		}

		// Get the current set of users (with their role information),
		// and create a map of them by id.
		currentUsers, err := context.datastore.GetUserInfoAll()
		if err != nil {
			context.Fatal(err)
			return
		}
		userMap := make(map[string]*datastore.User, len(currentUsers))
		for i, u := range currentUsers {
			userMap[u.Id] = &currentUsers[i]
		}

		// Create the set of deletable roles.
		roleSpecs := this.plan.Node().Roles()
		deleteRoleMap := make(map[datastore.Role]bool, len(roleSpecs))
		roleList := make([]datastore.Role, len(roleSpecs))
		for i, rs := range roleSpecs {
			var role datastore.Role
			role.Name = rs.Role
			role.Bucket = rs.Bucket
			deleteRoleMap[role] = true
			roleList[i] = role
		}

		// Get the list of all valid roles, and verify that the roles to be
		// deleted are proper.
		validRoles, err := context.datastore.GetRolesAll()
		if err != nil {
			context.Fatal(err)
			return
		}
		validKeyspaces, err := getAllKeyspaces(context.datastore)
		if err != nil {
			context.Fatal(err)
			return
		}
		err = validateRoles(roleList, validRoles, validKeyspaces)
		if err != nil {
			context.Fatal(err)
			return
		}

		// Since we only want to update each user once, even if the
		// statement mentions the user multiple times, create a map
		// of the input user ids.
		updateUsers := this.plan.Node().Users()
		updateUserIdMap := make(map[string]bool, len(updateUsers))
		for _, u := range updateUsers {
			updateUserIdMap[u] = true
		}

		for userId, _ := range updateUserIdMap {
			user := userMap[userId]
			if user == nil {
				context.Error(errors.NewUserNotFoundError(userId))
				continue
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
