//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"encoding/json"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

type GrantRole struct {
	statementBase

	roles     []string `json:"roles"`
	keyspaces []string `json:"keyspaces"`
	users     []string `json:"users"`
}

/*
The function NewGrantRole returns a pointer to the
GrantRole struct with the input argument values as fields.
*/
func NewGrantRole(roles []string, keyspaces []string, users []string) *GrantRole {
	rv := &GrantRole{
		roles:     roles,
		keyspaces: keyspaces,
		users:     users,
	}

	rv.stmt = rv
	return rv
}

/*
It calls the VisitGrantRole method by passing
in the receiver and returns the interface. It is a
visitor pattern.
*/
func (this *GrantRole) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitGrantRole(this)
}

/*
Returns nil.
*/
func (this *GrantRole) Signature() value.Value {
	return nil
}

/*
Returns nil.
*/
func (this *GrantRole) Formalize() error {
	return nil
}

/*
This method maps all the constituent clauses, namely the expression,
partition and where clause within a create index statement.
*/
func (this *GrantRole) MapExpressions(mapper expression.Mapper) (err error) {
	return nil
}

/*
Return expr from the statement.
*/
func (this *GrantRole) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *GrantRole) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	// Currently our privileges always attach to buckets. In this case,
	// the data being updated isn't a bucket, it's system security data,
	// so the code is leaving the bucket name blank.
	// This works because no bucket name is needed for this type of authorization.
	// If we absolutely had to provide a table name, it would make sense to use system:user_info,
	// because that's the virtual table where the data can be accessed.
	privs.Add("", auth.PRIV_SECURITY_WRITE)
	return privs, nil
}

/*
Returns the list of users to whom roles are being assigned.
*/
func (this *GrantRole) Users() []string {
	return this.users
}

/*
Returns the list of roles being assigned.
*/
func (this *GrantRole) Roles() []string {
	return this.roles
}

/*
Returns the list of keyspaces that qualify the roles being assigned.
*/
func (this *GrantRole) Keyspaces() []string {
	return this.keyspaces
}

/*
Marshals input receiver into byte array.
*/
func (this *GrantRole) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "grantRole"}
	r["users"] = this.users
	r["keyspaces"] = this.keyspaces
	r["roles"] = this.roles

	return json.Marshal(r)
}

func (this *GrantRole) Type() string {
	return "GRANT_ROLE"
}
