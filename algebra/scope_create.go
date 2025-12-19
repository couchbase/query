//  Copyright 2020-Present Couchbase, Inc.
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

/*
Represents the Create scope ddl statement. Type CreateScope is
a struct that contains fields mapping to each clause in the
create scope statement.
*/
type CreateScope struct {
	statementBase

	scope        *ScopeRef `json:"scope"`
	failIfExists bool      `json:"failIfExists"`
}

/*
The function NewCreateScope returns a pointer to the
CreateScope struct with the input argument values as fields.
*/
func NewCreateScope(scope *ScopeRef, failIfExists bool) *CreateScope {
	rv := &CreateScope{
		scope:        scope,
		failIfExists: failIfExists,
	}

	rv.stmt = rv
	return rv
}

/*
It calls the VisitCreateScope method by passing
in the receiver and returns the interface. It is a
visitor pattern.
*/
func (this *CreateScope) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateScope(this)
}

/*
Returns nil.
*/
func (this *CreateScope) Signature() value.Value {
	return nil
}

/*
Returns nil.
*/
func (this *CreateScope) Formalize() error {
	return nil
}

/*
This method maps all the constituent clauses.
*/
func (this *CreateScope) MapExpressions(mapper expression.Mapper) error {
	return nil
}

/*
Return expr from the create scope statement.
*/
func (this *CreateScope) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *CreateScope) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	fullName := this.scope.Path().BucketPath().FullName()
	privs.Add(fullName, auth.PRIV_QUERY_SCOPE_ADMIN, auth.PRIV_PROPS_NONE)

	return privs, nil
}

/*
Returns the scope reference of the scope to be created
*/
func (this *CreateScope) Scope() *ScopeRef {
	return this.scope
}

/*
Returns the name of the scope to be created
*/
func (this *CreateScope) Name() string {
	return this.scope.Path().Scope()
}

/*
Marshals input receiver into byte array.
*/
func (this *CreateScope) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "createScope"}
	r["scopeRef"] = this.scope
	r["failIfExists"] = this.failIfExists
	return json.Marshal(r)
}

func (this *CreateScope) Type() string {
	return "CREATE_SCOPE"
}

func (this *CreateScope) FailIfExists() bool {
	return this.failIfExists
}

func (this *CreateScope) String() string {
	var s strings.Builder
	s.WriteString("CREATE SCOPE ")

	if !this.failIfExists {
		s.WriteString("IF NOT EXISTS ")
	}

	s.WriteString(this.scope.path.ProtectedString())
	return s.String()
}
