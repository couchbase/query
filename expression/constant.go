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
	"github.com/couchbaselabs/query/value"
)

/*
The Constant type is a struct that comtains fields ExpressionBase
and a value. 
*/
type Constant struct {
	ExpressionBase
	value value.Value
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

/*
Create a new Constant out of an input value interface by
calling NewValue and setting the value component of the 
struct to it. Return a pointer to the Constant structure.
*/
func NewConstant(val interface{}) Expression {
	return &Constant{
		value: value.NewValue(val),
	}
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
A constant expression is indexable and hence this method
returns true.
*/
func (this *Constant) Indexable() bool {
	return true
}

/*
Check the type of the input expression. If it is a constant expression
call the Equals method for values over the receivers value and the
other expressions value. If not a constant then return false since
the expressions are not Equivalent.
*/
func (this *Constant) EquivalentTo(other Expression) bool {
	switch other := other.(type) {
	case *Constant:
		return this.value.Equals(other.value)
	default:
		return false
	}
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
This method is defined to access the value of the Constant 
expression. It returns the receivers value.
*/
func (this *Constant) Value() value.Value {
	return this.value
}
