//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/value"
)

/*
Represents the EXPLAIN FUNCTION statement
This statement is used to explain query statements inside UDFs
*/
type ExplainFunction struct {
	statementBase

	funcName functions.FunctionName `json:"name"`
}

/*
The function NewExplainFunction returns a pointer to the ExplainFunction struct with the input argument values as fields.
*/
func NewExplainFunction(funcName functions.FunctionName) *ExplainFunction {
	rv := &ExplainFunction{
		funcName: funcName,
	}

	rv.statementBase.stmt = rv
	return rv
}

/*
It calls the VisitExplainFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ExplainFunction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExplainFunction(this)
}

/*
This method returns the shape of the result, which is
a JSON string value.
*/
func (this *ExplainFunction) Signature() value.Value {
	return value.NewValue(value.JSON.String())
}

func (this *ExplainFunction) Formalize() error {
	return nil
}

// Returns nil since there are no expressions
func (this *ExplainFunction) MapExpressions(mapper expression.Mapper) error {
	return nil
}

// Returns nil since there are no expressions
func (this *ExplainFunction) Expressions() expression.Expressions {
	return nil
}

// Privileges can only be determined during execution phase
// Since it depends on the type of the function - Global or Scope
func (this *ExplainFunction) Privileges() (*auth.Privileges, errors.Error) {
	return auth.NewPrivileges(), nil
}

func (this *ExplainFunction) Type() string {
	return "EXPLAIN_FUNCTION"
}

func (this *ExplainFunction) FuncName() functions.FunctionName {
	return this.funcName
}
