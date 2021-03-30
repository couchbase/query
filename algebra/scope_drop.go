//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package algebra

import (
	"encoding/json"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
Represents the Drop scope ddl statement. Type DropScope is
a struct that contains fields mapping to each clause in the
drop scope statement.
*/
type DropScope struct {
	statementBase

	scope *ScopeRef `json:"scope"`
}

/*
The function NewDropScope returns a pointer to the
DropScope struct with the input argument values as fields.
*/
func NewDropScope(scope *ScopeRef) *DropScope {
	rv := &DropScope{
		scope: scope,
	}

	rv.stmt = rv
	return rv
}

/*
It calls the VisitDropScope method by passing in the
receiver and returns the interface. It is a visitor
pattern.
*/
func (this *DropScope) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropScope(this)
}

/*
Returns nil.
*/
func (this *DropScope) Signature() value.Value {
	return nil
}

/*
Returns nil.
*/
func (this *DropScope) Formalize() error {
	return nil
}

/*
Returns nil.
*/
func (this *DropScope) MapExpressions(mapper expression.Mapper) error {
	return nil
}

/*
Returns all contained Expressions.
*/
func (this *DropScope) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *DropScope) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	fullName := this.scope.Path().BucketPath().FullName()
	privs.Add(fullName, auth.PRIV_QUERY_BUCKET_ADMIN, auth.PRIV_PROPS_NONE)
	return privs, nil
}

/*
Returns the scope reference of the scope to be dropped
*/
func (this *DropScope) Scope() *ScopeRef {
	return this.scope
}

/*
Returns the name of the scope to be dropped
*/
func (this *DropScope) Name() string {
	return this.scope.Path().Scope()
}

/*
Marshals input receiver into byte array.
*/
func (this *DropScope) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "dropScope"}
	r["scopeRef"] = this.scope
	return json.Marshal(r)
}

func (this *DropScope) Type() string {
	return "DROP_SCOPE"
}
