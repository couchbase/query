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
	"strings"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type GrantRole struct {
	base
	plan *plan.GrantRole
}

func NewGrantRole(plan *plan.GrantRole, context *Context) *GrantRole {
	rv := &GrantRole{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *GrantRole) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitGrantRole(this)
}

func (this *GrantRole) Copy() Operator {
	rv := &GrantRole{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *GrantRole) PlanOp() plan.Operator {
	return this.plan
}

func validateRoles(candidateRoles, allRoles []datastore.Role) errors.Error {
	for _, candidate := range candidateRoles {
		foundMatch := false
		for _, permittedRole := range allRoles {
			if candidate.Name == permittedRole.Name {
				if candidate.Target == "" {
					if permittedRole.Target == "*" {
						return errors.NewRoleRequiresKeyspaceError(candidate.Name)
					}
				} else {
					if permittedRole.Target != "*" {
						return errors.NewRoleTakesNoKeyspaceError(candidate.Name)
					}

					// the keyspace is known to exist. no further checks needed
				}
				foundMatch = true
				break
			}
		}
		if !foundMatch {
			return errors.NewRoleNotFoundError(candidate.Name)
		}
	}
	return nil
}

type roleSource interface {
	Roles() []string
	Keyspaces() []*algebra.KeyspaceRef
}

func getRoles(node roleSource) ([]datastore.Role, errors.Error) {
	rolesList := auth.NormalizeRoleNames(node.Roles())
	keyspaceList := node.Keyspaces()

	if len(keyspaceList) == 0 {
		ret := make([]datastore.Role, len(rolesList))
		for i, v := range rolesList {
			ret[i].Name = v
		}
		return ret, nil
	} else {
		ret := make([]datastore.Role, 0, len(rolesList)*len(keyspaceList))
		for _, role := range rolesList {
			for _, ks := range keyspaceList {
				parts := ks.Path().Parts()
				if len(parts) != 3 {
					keyspace, err := datastore.GetKeyspace(parts...)
					if keyspace == nil {

						// we still want to be able to grant privileges on a bucket even
						// if it's missing a default collection
						if err != nil && len(parts) == 2 && err.Code() == errors.E_BUCKET_NO_DEFAULT_COLLECTION {
							bucket, _ := datastore.GetBucket(parts...)
							if bucket == nil {
								return nil, errors.NewNoSuchBucketError(ks.FullName())
							}
							ret = append(ret, datastore.Role{Name: role, Target: bucket.AuthKey()})
						} else {
							return nil, errors.NewNoSuchKeyspaceError(ks.FullName())
						}
					} else {
						ret = append(ret, datastore.Role{Name: role, Target: keyspace.AuthKey()})
					}
				} else {
					scope, _ := datastore.GetScope(parts...)
					if scope == nil {
						return nil, errors.NewNoSuchScopeError(ks.FullName())
					}
					ret = append(ret, datastore.Role{Name: role, Target: scope.AuthKey()})
				}
			}
		}
		return ret, nil
	}
}

// Retrieve the complete set of current users and their roles.
// Return them as a map indexed by domain:user_id.
func getUserMap(ds datastore.Datastore) (map[string]*datastore.User, errors.Error) {
	// Get the current set of users (with their role information),
	// and create a map of them by id.
	currentUsers, err := ds.GetUserInfoAll()
	if err != nil {
		return nil, err
	}
	userMap := make(map[string]*datastore.User, len(currentUsers))
	for i, u := range currentUsers {
		key := u.Domain + ":" + u.Id
		userMap[key] = &currentUsers[i]
	}
	return userMap, nil
}

// Given a string of users (in either plain user_id form or domain:user_id form),
// create a map of users, where all are in domain:user_id form.
func getUsersMap(users []string) map[string]bool {
	ret := make(map[string]bool, len(users))
	for _, v := range users {
		var domainUser string
		if strings.Contains(v, ":") {
			domainUser = v
		} else {
			domainUser = "local:" + v
		}
		ret[domainUser] = true
	}
	return ret
}

func (this *GrantRole) RunOnce(context *Context, parent value.Value) {
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

		// Create the set of new roles, in a form suitable for output.
		roleList, err := getRoles(this.plan.Node())
		if err != nil {
			context.Fatal(err)
			return
		}

		// Get the list of all valid roles, and verify that the roles to be granted are proper.
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

		if this.plan.Node().Groups() {
			this.grantGroupRoles(context, roleList)
		} else {
			this.grantUserRoles(context, roleList)
		}
	})
}

func (this *GrantRole) grantUserRoles(context *Context, roleList []datastore.Role) {

	userMap, err := getUserMap(context.datastore)
	if err != nil {
		context.Fatal(err)
		return
	}

	// Since we only want to update each user once, even if the statement mentions the user multiple times, create a map
	// of the input user ids in domain:user form.
	updateUserIdMap := getUsersMap(this.plan.Node().Users())

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
				context.Warning(errors.NewRoleAlreadyPresent("User", userId, auth.RoleToAlias(newRole.Name), newRole.Target))
				continue
			}
			user.Roles = append(user.Roles, newRole)
		}
		// Update the user with their new roles on the backend.
		user.Password = string([]byte{0}) // we are not including the password
		err = context.datastore.PutUserInfo(user)
		if err != nil {
			context.Error(err)
		}
	}
}

func getGroupMap(ds datastore.Datastore) (map[string]*datastore.Group, errors.Error) {
	// Get the current set of groups (with their role information) and create a map of them by id.
	currentGroups, err := ds.GetGroupInfoAll()
	if err != nil {
		return nil, err
	}
	groupMap := make(map[string]*datastore.Group, len(currentGroups))
	for i, g := range currentGroups {
		groupMap[g.Id] = &currentGroups[i]
	}
	return groupMap, nil
}

func getGroupsMap(groups []string) map[string]bool {
	ret := make(map[string]bool, len(groups))
	for _, v := range groups {
		ret[v] = true
	}
	return ret
}

func (this *GrantRole) grantGroupRoles(context *Context, roleList []datastore.Role) {

	groupMap, err := getGroupMap(context.datastore)
	if err != nil {
		context.Fatal(err)
		return
	}

	// Since we only want to update each group once, even if the statement mentions the group multiple times, create a map
	// of the input group ids
	updateGroupIdMap := getGroupsMap(this.plan.Node().Users()) // Users() is actually the list of groups when Groups() is true

	for groupId, _ := range updateGroupIdMap {
		group := groupMap[groupId]
		if group == nil {
			context.Error(errors.NewGroupNotFoundError(groupId))
			continue
		}
		// Add to the user the roles they do not already have.
		for _, newRole := range roleList {
			alreadyHasRole := false
			for _, existingRole := range group.Roles {
				if newRole == existingRole {
					alreadyHasRole = true
					break
				}
			}
			if alreadyHasRole {
				context.Warning(errors.NewRoleAlreadyPresent("Group", groupId, auth.RoleToAlias(newRole.Name), newRole.Target))
				continue
			}
			group.Roles = append(group.Roles, newRole)
		}
		// Update the group with their new roles on the backend.
		err = context.datastore.PutGroupInfo(group)
		if err != nil {
			context.Error(err)
		}
	}
}

func (this *GrantRole) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
