//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"strings"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/value"
)

/*
Represents the Drop function ddl statement. Type DropFunction is
a struct that contains fields mapping to each clause in the
create function statement. The fields just refer to the function name.
*/
type DropFunction struct {
	statementBase

	name            functions.FunctionName `json:"name"`
	failIfNotExists bool                   `json:"failIfNotExists"`
}

/*
The function NewDropFunction returns a pointer to the
DropFunction struct with the input argument values as fields.
*/
func NewDropFunction(name functions.FunctionName, failIfNotExists bool) *DropFunction {
	rv := &DropFunction{
		name:            name,
		failIfNotExists: failIfNotExists,
	}

	rv.stmt = rv
	return rv
}

func (this *DropFunction) Name() functions.FunctionName {
	return this.name
}

func (this *DropFunction) FailIfNotExists() bool {
	return this.failIfNotExists
}

/*
It calls the VisitDropFunction method by passing
in the receiver and returns the interface. It is a
visitor pattern.
*/
func (this *DropFunction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropFunction(this)
}

/*
Returns nil.
*/
func (this *DropFunction) Signature() value.Value {
	return nil
}

/*
Returns nil.
*/
func (this *DropFunction) Formalize() error {
	return nil
}

/*
This method maps all the constituent clauses, but here none have expressions
*/
func (this *DropFunction) MapExpressions(mapper expression.Mapper) (err error) {
	return
}

/*
Return expr from the create function statement.
*/
func (this *DropFunction) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *DropFunction) Privileges() (*auth.Privileges, errors.Error) {
	// we can't determine privileges here because we need to know the
	// function type, which we get reliably only at execution time
	return auth.NewPrivileges(), nil
}

func (this *DropFunction) Type() string {
	return "DROP_FUNCTION"
}

func (this *DropFunction) String() string {
	var s strings.Builder
	s.WriteString("DROP FUNCTION ")
	if !this.failIfNotExists {
		s.WriteString("IF EXISTS ")
	}
	s.WriteString(this.name.ProtectedKey())
	return s.String()
}
