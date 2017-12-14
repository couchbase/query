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
	val, ok := context.(Context).NamedArg(this.name)

	if ok {
		return val, nil
	} else {
		return nil, fmt.Errorf("No value for named parameter $%s.", this.name)
	}
}

/*
Returns input receiver Named parameter.
*/
func (this *NamedParameter) Static() expression.Expression {
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
