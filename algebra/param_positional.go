//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"fmt"

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
	val, ok := context.(Context).PositionalArg(this.position)

	if ok {
		return val, nil
	} else {
		return nil, fmt.Errorf("No value for positional parameter $%d.", this.position)
	}
}

/*
Returns input receiver positional parameter.
*/
func (this *PositionalParameter) Static() expression.Expression {
	return this
}

/*
Returns false. Not indexable.
*/
func (this *PositionalParameter) Indexable() bool {
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
