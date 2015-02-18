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
	"math"

	"github.com/couchbase/query/value"
)

/*
Nested expressions are used to access elements inside of arrays.
They support using the bracket notation ([position]) to access
elements inside an array. Type Element is a struct that implements
BinaryFunctionBase.
*/
type Element struct {
	BinaryFunctionBase
}

/*
The function NewElement calls NewBinaryFunctionBase to define the
field with input operand expressions first and second, as input.
*/

func NewElement(first, second Expression) *Element {
	rv := &Element{
		*NewBinaryFunctionBase("element", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitElement method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Element) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitElement(this)
}

/*
It returns a value type JSON.
*/
func (this *Element) Type() value.Type { return value.JSON }

/*
Calls the Eval method for Binary functions and passes in the
receiver, current item and current context.
*/
func (this *Element) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method evaluates the element using the first and second value
and returns the result value. If the second operand type is missing
then return a missing value. If it is a number, check if it is an
absolute number (equal to its trucated value), and return the element
at that index using the Index method. If it isnt a number or missing,
then check the first elements type. If it is missing return missing
otherwise return null value.
*/
func (this *Element) Apply(context Context, first, second value.Value) (value.Value, error) {
	switch second.Type() {
	case value.NUMBER:
		s := second.Actual().(float64)
		if s == math.Trunc(s) {
			v, _ := first.Index(int(s))
			return v, nil
		}
	case value.MISSING:
		return value.MISSING_VALUE, nil
	}

	if first.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else {
		return value.NULL_VALUE, nil
	}
}

/*
The constructor returns a NewElement with the operands
cast to a Function as the FunctionConstructor.
*/
func (this *Element) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewElement(operands[0], operands[1])
	}
}

/*
Set value at index. Evaluate the first and second operands.
If the second is an abosulte number(integer) then call
SetIndex method to set that index in the first operand with
value val. If the SetIndex method is successful return true.
For all other cases return false.
*/
func (this *Element) Set(item, val value.Value, context Context) bool {
	second, er := this.Second().Evaluate(item, context)
	if er != nil {
		return false
	}

	first, er := this.First().Evaluate(item, context)
	if er != nil {
		return false
	}

	switch second.Type() {
	case value.NUMBER:
		s := second.Actual().(float64)
		if s == math.Trunc(s) {
			er := first.SetIndex(int(s), val)
			return er == nil
		}
	}

	return false
}

/*
Return false.
*/
func (this *Element) Unset(item value.Value, context Context) bool {
	return false
}
