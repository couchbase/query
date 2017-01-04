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

type GrantRole struct {
	base
	plan *plan.GrantRole
}

func NewGrantRole(plan *plan.GrantRole) *GrantRole {
	rv := &GrantRole{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *GrantRole) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitGrantRole(this)
}

func (this *GrantRole) Copy() Operator {
	return &GrantRole{this.base.copy(), this.plan}
}

func (this *GrantRole) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover()       // Recover from any panic
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

		// Create the set of new roles, in a form suitable for output.
		roleSpecs := this.plan.Node().Roles()
		roleList := make([]datastore.Role, len(roleSpecs))
		for i, rs := range roleSpecs {
			roleList[i].Name = rs.Role
			roleList[i].Bucket = rs.Bucket
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
			// Add to the user the roles they do not already have.
			for _, newRole := range roleList {
				alreadyHasRole := false
				for _, existingRole := range user.Roles {
					if newRole == existingRole {
						alreadyHasRole = true
						break
					}
				}
				if alreadyHasRole {
					continue
				}
				user.Roles = append(user.Roles, newRole)
			}
			// Update the user with their new roles on the backend.
			err = context.datastore.PutUserInfo(user)
			if err != nil {
				context.Error(err)
			}
		}

	})
}

func (this *GrantRole) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
