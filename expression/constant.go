//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

import (
	"github.com/couchbase/query/value"
)

type Constant struct {
	ExpressionBase
	value value.Value // Overshadows ExpressionBase.value
}

/*
Pre-define commonly used constant expressions.
*/
var NULL_EXPR = NewConstant(value.NULL_VALUE)
var MISSING_EXPR = NewConstant(value.MISSING_VALUE)
var FALSE_EXPR = NewConstant(value.FALSE_VALUE)
var TRUE_EXPR = NewConstant(value.TRUE_VALUE)
var ZERO_EXPR = NewConstant(value.ZERO_VALUE)
var ONE_EXPR = NewConstant(value.ONE_VALUE)
var EMPTY_STRING_EXPR = NewConstant(value.EMPTY_STRING_VALUE)
var EMPTY_ARRAY_EXPR = NewConstant(value.EMPTY_ARRAY_VALUE)

func NewConstant(val interface{}) Expression {
	rv := &Constant{
		value: value.NewValue(val),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Constant) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitConstant(this)
}

func (this *Constant) Type() value.Type { return this.value.Type() }

func (this *Constant) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.value, nil
}

func (this *Constant) Value() value.Value {
	return this.value
}

/*
Returns this constant expression.
*/
func (this *Constant) Static() Expression {
	return this
}

/*
A constant expression is indexable as part of another expression.
*/
func (this *Constant) Indexable() bool {
	return true
}

/*
Indicates if this expression is equivalent to the other expression.
False negatives are allowed.
*/
func (this *Constant) EquivalentTo(other Expression) bool {
	switch other := other.(type) {
	case *FieldName:
		return !other.caseInsensitive && (this.value == other.value)
	default:
		return this.valueEquivalentTo(other)
	}
}

func (this *Constant) DependsOn(other Expression) bool {
	return false
}

func (this *Constant) CoveredBy(keyspace string, exprs Expressions, single bool) Covered {
	return CoveredTrue
}

func (this *Constant) Children() Expressions {
	return nil
}

func (this *Constant) MapChildren(mapper Mapper) error {
	return nil
}

/*
Constants are not transformed, so no need to copy.
*/
func (this *Constant) Copy() Expression {
	return this
}

func (this *Constant) SurvivesGrouping(groupKeys Expressions, allowed *value.ScopeValue) (
	bool, Expression) {
	return true, nil
}
