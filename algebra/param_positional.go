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
A positional parameter uses a position of the parameter in
the query. Type PositionalParameter is a struct that inherits
from ExpressionBase and contains a field position representing
the param position.
*/
type PositionalParameter struct {
	expression.ExpressionBase
	position int
}

/*
The function NewPositionalParameter returns a pointer
to the PositionalParameter struct with the input
argument position as a field.
*/
func NewPositionalParameter(position int) expression.Expression {
	rv := &PositionalParameter{
		position: position,
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitPositionalParameter method by passing in
the receiver and returns the interface. It is a visitor pattern.
*/
func (this *PositionalParameter) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitPositionalParameter(this)
}

/*
Returns a JSON value.
*/
func (this *PositionalParameter) Type() value.Type { return value.JSON }

/*
Evaluate the input Positional Parameter and return the
value.
*/
func (this *PositionalParameter) Evaluate(item value.Value, context expression.Context) (
	value.Value, error) {
	if context == nil {
		return nil, errors.NewNilEvaluateParamError("context")
	}
	val, ok := context.(Context).PositionalArg(this.position)

	if ok {
		return val, nil
	} else {
		return nil, fmt.Errorf("No value for positional parameter $%d%v.", this.position, this.ErrorContext())
	}
}

/*
Returns input receiver positional parameter.
*/
func (this *PositionalParameter) Static() expression.Expression {
	return this
}

func (this *PositionalParameter) StaticNoVariable() expression.Expression {
	return this
}

/*
Returns false. Not indexable.
*/
func (this *PositionalParameter) Indexable() bool {
	return false
}

/*
Returns false. Not IndexAggregatable.
*/
func (this *PositionalParameter) IndexAggregatable() bool {
	return false
}

/*
Checks if receiver and input expression are equivalent. If the input
expression is a positional parameter check if the two positions are
equal.
*/
func (this *PositionalParameter) EquivalentTo(other expression.Expression) bool {
	switch other := other.(type) {
	case *PositionalParameter:
		return this.position == other.position
	default:
		return false
	}
}

/*
Calls the EquivalentTo method.
*/
func (this *PositionalParameter) SubsetOf(other expression.Expression) bool {
	return this.EquivalentTo(other)
}

/*
Returns nil.
*/
func (this *PositionalParameter) Children() expression.Expressions {
	return nil
}

/*
Returns nil.
*/
func (this *PositionalParameter) MapChildren(mapper expression.Mapper) error {
	return nil
}

/*
Returns receiver.
*/
func (this *PositionalParameter) Copy() expression.Expression {
	return this
}

func (this *PositionalParameter) SurvivesGrouping(groupKeys expression.Expressions, allowed *value.ScopeValue) (
	bool, expression.Expression) {
	return true, nil
}

/*
Returns the position.
*/
func (this *PositionalParameter) Position() int {
	return this.position
}
