//  Copyright 2014-Present Couchbase, Inc.
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
	"github.com/couchbase/query/value"
)

/*
Represents the Execute command. The argument to EXECUTE must
evaluate to a string.
Type Execute is a struct that contains a json object value that represents
a plan.Prepared.
*/
type Execute struct {
	statementBase

	prepared     value.Value `json:"prepared"`
	preparedName string

	// this contains either named parameters (a map of values)
	// or positional (a slice)
	using expression.Expression `json:"using"`
}

/*
The function NewExecute returns a pointer to the Execute
struct with the input argument expressions value as a field.
*/
func NewExecute(prepared expression.Expression, using expression.Expression) *Execute {
	var preparedValue value.Value
	var preparedString string

	switch prepared := prepared.(type) {
	case *expression.Identifier:
		preparedString = prepared.Alias()
		preparedValue = value.NewValue(preparedString)
	default:
		preparedValue = prepared.Value()
		if preparedValue != nil {
			if preparedValue.Type() == value.STRING {
				preparedString = preparedValue.Actual().(string)
			} else {
				preparedString = preparedValue.String()
			}
		}
	}

	rv := &Execute{
		prepared:     preparedValue,
		preparedName: preparedString,
		using:        using,
	}

	rv.stmt = rv
	return rv
}

/*
It calls the VisitExecute method by passing in the receiver
and returns the interface. It is a visitor pattern.
*/
func (this *Execute) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExecute(this)
}

/*
This method returns the shape of the result, which is
unkown at this time and will be evaluated at execution time
*/
func (this *Execute) Signature() value.Value {
	return nil
}

/*
It's an execute
*/
func (this *Execute) Type() string {
	return "EXECUTE"
}

/*
Returns nil.
*/
func (this *Execute) Formalize() error {
	return nil
}

/*
Returns nil.
*/
func (this *Execute) MapExpressions(mapper expression.Mapper) error {
	return nil
}

/*
Returns all contained Expressions.
*/
func (this *Execute) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *Execute) Privileges() (*auth.Privileges, errors.Error) {
	return nil, nil
}

/*
Returns the input prepared name that represents the prepared
statement.
*/
func (this *Execute) Prepared() string {
	return this.preparedName
}

/*
Returns the input prepared value that represents the prepared
statement.
*/
func (this *Execute) PreparedValue() value.Value {
	return this.prepared
}

/*
Returns the input placeholder values
*/
func (this *Execute) Using() expression.Expression {
	return this.using
}

func (this *Execute) String() string {
	var s strings.Builder
	s.WriteString("EXECUTE ")
	s.WriteRune('`')
	s.WriteString(this.preparedName)
	s.WriteRune('`')
	if this.using != nil {
		s.WriteString(" USING ")
		s.WriteString(this.using.String())
	}
	return s.String()
}
