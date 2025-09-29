//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"encoding/json"
	"strings"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

type GrantRole struct {
	statementBase

	roles     []string       `json:"roles"`
	keyspaces []*KeyspaceRef `json:"keyspaces"`
	users     []string       `json:"users"`
	groups    bool           `json:"groups"`
}

/*
The function NewGrantRole returns a pointer to the
GrantRole struct with the input argument values as fields.
*/
func NewGrantRole(roles []string, keyspaces []*KeyspaceRef, users []string, groups bool) *GrantRole {
	rv := &GrantRole{
		roles:     roles,
		keyspaces: keyspaces,
		users:     users,
		groups:    groups,
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
	privs.Add("", auth.PRIV_SECURITY_WRITE, auth.PRIV_PROPS_NONE)
	return privs, nil
}

func (this *GrantRole) Groups() bool {
	return this.groups
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
func (this *GrantRole) Keyspaces() []*KeyspaceRef {
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
	r["groups"] = this.groups

	return json.Marshal(r)
}

func (this *GrantRole) Type() string {
	return "GRANT_ROLE"
}

func (this *GrantRole) String() string {
	var s strings.Builder
	s.WriteString("GRANT ")
	for i, role := range this.roles {
		if i > 0 {
			s.WriteString(", ")
		}
		s.WriteRune('`')
		s.WriteString(role)
		s.WriteRune('`')
	}

	if len(this.keyspaces) > 0 {
		s.WriteString(" ON ")
		for i, keyspace := range this.keyspaces {
			if i > 0 {
				s.WriteString(", ")
			}
			s.WriteString(keyspace.Path().ProtectedString())
		}
	}

	s.WriteString(" TO")

	if !this.groups {
		s.WriteString(" USERS ")
	} else {
		s.WriteString(" GROUPS ")
	}

	for i, user := range this.users {
		if i > 0 {
			s.WriteString(", ")
		}

		if this.groups {
			s.WriteRune('`')
			s.WriteString(user)
			s.WriteRune('`')
		} else {
			s.WriteString(DecodeUsername(user))
		}
	}

	return s.String()
}
