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

// Source type constants for role grant/revoke
const (
	SOURCE_KEYSPACE        = ""
	SOURCE_CATALOG         = "catalog"
	SOURCE_CREDENTIALSTORE = "credentialstore"
)

type GrantRole struct {
	statementBase

	roles      []string       `json:"roles"`
	keyspaces  []*KeyspaceRef `json:"keyspaces"`
	users      []string       `json:"users"`
	groups     bool           `json:"groups"`
	sourceType string         `json:"sourceType"`
}

/*
The function NewGrantRole returns a pointer to the
GrantRole struct with the input argument values as fields.
*/
// NewGrantRoleInfer constructs a GrantRole for the no-ON-clause form, inferring
// the source type and wildcard keyspace target from the role names.
func NewGrantRoleInfer(roles []string, users []string, groups bool) *GrantRole {
	sourceType, needsTarget := SourceTypeFromRoles(roles)
	var keyspaces []*KeyspaceRef
	if needsTarget {
		keyspaces = []*KeyspaceRef{NewKeyspaceRefWithContext("*", "", "", "")}
	}
	return NewGrantRole(roles, keyspaces, users, groups, sourceType)
}

func NewGrantRole(roles []string, keyspaces []*KeyspaceRef, users []string, groups bool, sourceType string) *GrantRole {
	rv := &GrantRole{
		roles:      roles,
		keyspaces:  keyspaces,
		users:      users,
		groups:     groups,
		sourceType: sourceType,
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
	privs.Add("", auth.PRIV_USERS_WRITE, auth.PRIV_PROPS_NONE)
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
Returns the source type for the roles being assigned.
*/
func (this *GrantRole) SourceType() string {
	return this.sourceType
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
	if this.sourceType != "" {
		r["sourceType"] = this.sourceType
	}

	return json.Marshal(r)
}

// SourceTypeFromRoles infers the source type from a list of role names and whether a target (ON clause) is required.
// The bool is true when all roles have the _external_catalog suffix or are credential_consumer.
// _external_catalog suffix roles must not be mixed with external_catalog_admin / external_catalog_reader.
// Mixed source types return SOURCE_KEYSPACE, false.
func SourceTypeFromRoles(roles []string) (string, bool) {
	roleSourceType := func(lc string) string {
		if strings.HasSuffix(lc, "_external_catalog") || lc == "external_catalog_admin" || lc == "external_catalog_reader" {
			return SOURCE_CATALOG
		}
		if lc == "credential_consumer" {
			return SOURCE_CREDENTIALSTORE
		}
		return SOURCE_KEYSPACE
	}

	if len(roles) == 0 {
		return SOURCE_KEYSPACE, false
	}

	var hasSuffixCatalog, hasAdminOrReader bool
	lc0 := strings.ToLower(roles[0])
	result := roleSourceType(lc0)

	for _, role := range roles {
		lc := strings.ToLower(role)
		if roleSourceType(lc) != result {
			return SOURCE_KEYSPACE, false
		}
		if strings.HasSuffix(lc, "_external_catalog") {
			hasSuffixCatalog = true
		} else if lc == "external_catalog_admin" || lc == "external_catalog_reader" {
			hasAdminOrReader = true
		}
	}

	// _external_catalog suffix roles must not be combined with external_catalog_admin/reader
	if hasSuffixCatalog && hasAdminOrReader {
		return SOURCE_KEYSPACE, false
	}

	needsTarget := hasSuffixCatalog || result == SOURCE_CREDENTIALSTORE
	return result, needsTarget
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
		if this.sourceType != "" {
			s.WriteRune(' ')
			s.WriteString(strings.ToUpper(this.sourceType))
		}
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
