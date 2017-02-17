//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

import (
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
// DoAdmin
//
///////////////////////////////////////////////////

/*
Do one of a number of administrative functions.
The input is a JSON object. The format of the object varies by the action taken.

For the GRANT_ROLE action, paralleling the GRANT ROLE statement, the input is of this form:

    {
      "action" : "GRANT_ROLE",
      "users" : [ "aaron", "robert", "callie" ],
      "roles" : [ { "name" : "cluster_admin" },
                  { "name" : "data_reader", "bucket" : "testbucket" } ]
    }

The "users" and "roles" arrays can each be replaced by a single entry,
using the "user" and "role" fields, respectively:

    {
      "action" : "GRANT_ROLE",
      "user" : "aaron",
      "role" : { "name" : "data_reader", "bucket" : "testbucket" }
    }

The function returns an object like this on success, else throws an error:

    {
      "status" : "success"
    }
*/

type DoAdmin struct {
	UnaryFunctionBase
}

func NewDoAdmin(operand Expression) Function {
	rv := &DoAdmin{
		*NewUnaryFunctionBase("do_admin", operand),
	}

	rv.expr = rv
	return rv
}

func DoGrantRole(users []string, roles []*auth.Role, context Context) errors.Error {
	// Get the current set of users (with their role information),
	// and create a map of them by id.
	currentUsers, err := context.GetUserInfoAll()
	if err != nil {
		return err
	}
	userMap := make(map[string]*auth.User, len(currentUsers))
	for i, u := range currentUsers {
		userMap[u.Id] = &currentUsers[i]
	}

	// Since we only want to update each user once, even if the
	// statement mentions the user multiple times, create a map
	// of the input user ids.
	updateUserIdMap := make(map[string]bool, len(users))
	for _, u := range users {
		updateUserIdMap[u] = true
	}
	for userId, _ := range updateUserIdMap {
		user := userMap[userId]
		if user == nil {
			return errors.NewUserNotFoundError(userId)
		}
		// Add to the user the roles they do not already have.
		for _, newRole := range roles {
			alreadyHasRole := false
			for _, existingRole := range user.Roles {
				if *newRole == existingRole {
					alreadyHasRole = true
					break
				}
			}
			if alreadyHasRole {
				continue
			}
			user.Roles = append(user.Roles, *newRole)
		}
		// Update the user with their new roles on the backend.
		err = context.PutUserInfo(user)
		if err != nil {
			return err
		}
	}
	return nil
}

func parseUser(user interface{}) (string, error) {
	userVal := value.NewValue(user)
	if userVal.Type() != value.STRING {
		return "", errors.NewGrantRoleUserMustBeString(userVal)
	}
	return userVal.Actual().(string), nil
}

func parseRole(role interface{}) (*auth.Role, error) {
	roleVal := value.NewValue(role)
	if roleVal.Type() != value.OBJECT {
		return nil, errors.NewGrantRoleRoleMustBeObject(role)
	}
	name, has_name := roleVal.Field("name")
	bucket, has_bucket := roleVal.Field("bucket")

	if !has_name {
		return nil, errors.NewGrantRoleRoleNameMustBePresent(role)
	}
	nameVal := value.NewValue(name)
	if nameVal.Type() != value.STRING {
		return nil, errors.NewGrantRoleFieldMustBeString("name", name)
	}
	nameString := nameVal.Actual().(string)

	bucketString := ""
	if has_bucket {
		bucketVal := value.NewValue(bucket)
		if bucketVal.Type() != value.STRING {
			return nil, errors.NewGrantRoleFieldMustBeString("bucket", bucket)
		}
		bucketString = bucketVal.Actual().(string)
	}

	return &auth.Role{Name: nameString, Keyspace: bucketString}, nil
}

func doGrantRoleAction(input map[string]interface{}, context Context) error {
	// Get the users.
	user, has_user := input["user"]
	users, has_users := input["users"]
	if (has_user && has_users) || (!has_user && !has_users) {
		// User or users, not both, not neither.
		return errors.NewGrantRoleHasUserOrUsers(input)
	}
	var usersList []string
	if has_users {
		usersVal := value.NewValue(users)
		if usersVal.Type() != value.ARRAY {
			return errors.NewGrantRoleUsersMustBeArray(users)
		}
		inputUsers := usersVal.Actual().([]interface{})
		usersList = make([]string, len(inputUsers))
		for i, inputUser := range inputUsers {
			parsedUser, err := parseUser(inputUser)
			if err != nil {
				return err
			}
			usersList[i] = parsedUser
		}
	}
	if has_user {
		usersList = make([]string, 1)
		parsedUser, err := parseUser(user)
		if err != nil {
			return err
		}
		usersList[0] = parsedUser
	}

	// Get the roles
	role, has_role := input["role"]
	roles, has_roles := input["roles"]
	if (has_role && has_roles) || (!has_role && !has_roles) {
		// Role or roles, not both, not neither.
		return errors.NewGrantRoleHasRoleOrRoles(input)
	}
	var rolesList []*auth.Role
	if has_roles {
		rolesVal := value.NewValue(roles)
		if rolesVal.Type() != value.ARRAY {
			return errors.NewGrantRoleRolesMustBeArray(roles)
		}
		inputRoles := rolesVal.Actual().([]interface{})
		rolesList = make([]*auth.Role, len(inputRoles))
		for i, inputRole := range inputRoles {
			parsedRole, err := parseRole(inputRole)
			if err != nil {
				return err
			}
			rolesList[i] = parsedRole
		}
	}
	if has_role {
		rolesList = make([]*auth.Role, 1)
		parsedRole, err := parseRole(role)
		if err != nil {
			return err
		}
		rolesList[0] = parsedRole
	}

	// Invoke the action.
	return DoGrantRole(usersList, rolesList, context)
}

/*
Visitor pattern.
*/
func (this *DoAdmin) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DoAdmin) Type() value.Type { return value.OBJECT }

func (this *DoAdmin) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *DoAdmin) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	// Parse the input.
	inputMap, ok := arg.Actual().(map[string]interface{})
	if !ok {
		return nil, errors.NewAdminInputNotObject(arg.Actual())
	}
	action, present := inputMap["action"]
	if !present {
		return nil, errors.NewAdminActionNotPresent(inputMap)
	}
	actionVal := value.NewValue(action)
	if actionVal.Type() != value.STRING {
		return nil, errors.NewAdminActionMustBeString(action)
	}
	var err error
	actionString := actionVal.String()
	if actionString == "\"GRANT_ROLE\"" {
		err = doGrantRoleAction(inputMap, context)
	} else {
		return nil, errors.NewAdminUnknownAction(actionString)
	}

	if err != nil {
		return nil, err
	}

	retMap := make(map[string]interface{}, 1)
	retMap["result"] = "success"
	return value.NewValue(retMap), nil
}

/*
Factory method pattern.
*/
func (this *DoAdmin) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewDoAdmin(operands[0])
	}
}
