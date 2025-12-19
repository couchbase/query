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
Represents the Execute function ddl statement. Type ExecuteFunction is
a struct that contains fields mapping to each clause in the
create function statement. The fields refer to the function name
and argument expression list
*/
type ExecuteFunction struct {
	statementBase

	name  functions.FunctionName `json:"name"`
	exprs expression.Expressions `json:"expressions"`
}

/*
The function NewExecuteFunction returns a pointer to the
ExecuteFunction struct with the input argument values as fields.
*/
func NewExecuteFunction(name functions.FunctionName, exprs expression.Expressions) *ExecuteFunction {
	rv := &ExecuteFunction{
		name:  name,
		exprs: exprs,
	}

	rv.stmt = rv
	return rv
}

func (this *ExecuteFunction) Name() functions.FunctionName {
	return this.name
}

/*
It calls the VisitExecuteFunction method by passing
in the receiver and returns the interface. It is a
visitor pattern.
*/
func (this *ExecuteFunction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExecuteFunction(this)
}

/*
Returns nil.
*/
func (this *ExecuteFunction) Signature() value.Value {
	return nil
}

/*
Returns nil.
*/
func (this *ExecuteFunction) Formalize() error {
	return this.MapExpressions(expression.NewFormalizer("", nil))
}

/*
This method maps all the constituent clauses, namely the argument expression list
*/
func (this *ExecuteFunction) MapExpressions(mapper expression.Mapper) (err error) {
	if len(this.exprs) > 0 {
		err = this.exprs.MapExpressions(mapper)
		if err != nil {
			return
		}
	}
	return
}

/*
Return expr from the create function statement.
*/
func (this *ExecuteFunction) Expressions() expression.Expressions {
	return this.exprs
}

/*
Returns all required privileges.
*/
func (this *ExecuteFunction) Privileges() (*auth.Privileges, errors.Error) {
	// we can't determine privileges here because we need to know the
	// function type, which we get reliably only at execution time
	return auth.NewPrivileges(), nil
}

func (this *ExecuteFunction) Type() string {
	return "EXECUTE_FUNCTION"
}

func (this *ExecuteFunction) String() string {
	var s strings.Builder
	s.WriteString("EXECUTE FUNCTION ")
	s.WriteString(this.name.ProtectedKey())
	if len(this.exprs) > 0 {
		s.WriteString(" (")
		for i, expr := range this.exprs {
			if i > 0 {
				s.WriteString(", ")
			}
			s.WriteString(expr.String())
		}
		s.WriteString(")")
	}
	return s.String()
}
