//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package algebra

import (
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/value"
)

/*
Represents the Create function ddl statement. Type CreateFunction is
a struct that contains fields mapping to each clause in the
create function statement. The fields refer to the function name and
and function body
*/
type CreateFunction struct {
	statementBase

	name    functions.FunctionName `json:"name"`
	body    functions.FunctionBody `json:"body"`
	replace bool                   `json:"replace"`
}

/*
The function NewCreateFunction returns a pointer to the
CreateFunction struct with the input argument values as fields.
*/
func NewCreateFunction(name functions.FunctionName, body functions.FunctionBody, replace bool) *CreateFunction {
	rv := &CreateFunction{
		name:    name,
		body:    body,
		replace: replace,
	}

	rv.stmt = rv
	return rv
}

func (this *CreateFunction) Name() functions.FunctionName {
	return this.name
}

func (this *CreateFunction) Body() functions.FunctionBody {
	return this.body
}

func (this *CreateFunction) Replace() bool {
	return this.replace
}

/*
It calls the VisitCreateFunction method by passing
in the receiver and returns the interface. It is a
visitor pattern.
*/
func (this *CreateFunction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateFunction(this)
}

/*
Returns nil.
*/
func (this *CreateFunction) Signature() value.Value {
	return nil
}

/*
Returns nil.
*/
func (this *CreateFunction) Formalize() error {
	return nil
}

/*
This method maps all the constituent clauses, but here non have expressions
*/
func (this *CreateFunction) MapExpressions(mapper expression.Mapper) (err error) {
	return
}

/*
Return expr from the create function statement.
*/
func (this *CreateFunction) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *CreateFunction) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	priv := functions.GetPrivilege(this.name, this.body)
	privs.Add(this.name.Key(), priv, auth.PRIV_PROPS_NONE)

	return privs, nil
}

func (this *CreateFunction) Type() string {
	return "CREATE_FUNCTION"
}
