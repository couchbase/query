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

/*
The Constant type is a struct that inherits from ExpressionBase
and contains a value.
*/
type Constant struct {
	ExpressionBase
	value value.Value // Overshadows ExpressionBase.value
}

/*
Define a set of constant expressions created from values for null,
missing, true, false, zero and one.
*/
var NULL_EXPR = NewConstant(value.NULL_VALUE)
var MISSING_EXPR = NewConstant(value.MISSING_VALUE)
var FALSE_EXPR = NewConstant(value.FALSE_VALUE)
var TRUE_EXPR = NewConstant(value.TRUE_VALUE)
var ZERO_EXPR = NewConstant(value.ZERO_VALUE)
var ONE_EXPR = NewConstant(value.ONE_VALUE)
var EMPTY_STRING_EXPR = NewConstant(value.EMPTY_STRING_VALUE)
var EMPTY_ARRAY_EXPR = NewConstant(value.EMPTY_ARRAY_VALUE)

/*
Create a new Constant out of an input value interface by
calling NewValue and setting the value component of the
struct to it. Return a pointer to the Constant structure.
*/
func NewConstant(val interface{}) Expression {
	rv := &Constant{
		value: value.NewValue(val),
	}

	rv.expr = rv
	return rv
}

/*
Takes an input Visitor and calls VisitConstant on the receiver.
*/
func (this *Constant) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitConstant(this)
}

/*
Return the type of the value component of the struct.
*/
func (this *Constant) Type() value.Type { return this.value.Type() }

/*
Evaluates to the value component of the receiver. It returns nil as the
error.
*/
func (this *Constant) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.value, nil
}

/*
This method is defined to access the value of the Constant
expression. It returns the receivers value.
*/
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
	return this.ValueEquals(other)
}

/*
Constant expressions do not have children. Hence return nil.
*/
func (this *Constant) Children() Expressions {
	return nil
}

/*
Return nil.
*/
func (this *Constant) MapChildren(mapper Mapper) error {
	return nil
}

/*
Constants are not transformed, so no need to copy.
*/
func (this *Constant) Copy() Expression {
	return this
}
