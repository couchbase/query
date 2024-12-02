//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"fmt"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
The named parameter is specified using formal param names
in a query. The main advantage of a named parameter is
that we dont have to remember the position of the parameter.
Type NamedParameter is a struct that inherits from
ExpressionBase and contains a field name representing the
param name.
*/
type NamedParameter struct {
	expression.ExpressionBase
	name string
}

/*
The function NewNamedParameter returns a pointer to the
NamedParameter struct with the input argument name as a
field.
*/
func NewNamedParameter(name string) expression.Expression {
	rv := &NamedParameter{
		name: name,
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitNamedParameter method by passing in
the receiver and returns the interface. It is a visitor pattern.
*/
func (this *NamedParameter) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitNamedParameter(this)
}

/*
Returns a JSON value.
*/
func (this *NamedParameter) Type() value.Type { return value.JSON }

/*
Evaluate the input Named Parameter and return the
value.
*/
func (this *NamedParameter) Evaluate(item value.Value, context expression.Context) (
	value.Value, error) {
	if context == nil {
		return nil, errors.NewNilEvaluateParamError("context")
	}
	val, ok := context.(Context).NamedArg(this.name)

	if ok {
		return val, nil
	} else {
		return nil, fmt.Errorf("No value for named parameter $%s%v.", this.name, this.ErrorContext())
	}
}

/*
Returns input receiver Named parameter.
*/
func (this *NamedParameter) Static() expression.Expression {
	return this
}

func (this *NamedParameter) StaticNoVariable() expression.Expression {
	return this
}

/*
Returns false. Not indexable.
*/
func (this *NamedParameter) Indexable() bool {
	return false
}

/*
Returns false. Not IndexAggregatable.
*/
func (this *NamedParameter) IndexAggregatable() bool {
	return false
}

/*
Checks if receiver and input expression are equivalent. If the input
expression is a named parameter check if the two name strings are
equal.
*/
func (this *NamedParameter) EquivalentTo(other expression.Expression) bool {
	switch other := other.(type) {
	case *NamedParameter:
		return this.name == other.name
	default:
		return false
	}
}

/*
Calls the EquivalentTo method.
*/
func (this *NamedParameter) SubsetOf(other expression.Expression) bool {
	return this.EquivalentTo(other)
}

/*
Returns nil.
*/
func (this *NamedParameter) Children() expression.Expressions {
	return nil
}

/*
Returns nil.
*/
func (this *NamedParameter) MapChildren(mapper expression.Mapper) error {
	return nil
}

/*
Returns receiver.
*/
func (this *NamedParameter) Copy() expression.Expression {
	return this
}

func (this *NamedParameter) SurvivesGrouping(groupKeys expression.Expressions, allowed *value.ScopeValue) (
	bool, expression.Expression) {
	return true, nil
}

/*
Returns name.
*/
func (this *NamedParameter) Name() string {
	return this.name
}
