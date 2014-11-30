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
Represents Div for arithmetic expressions. Type Div is a struct
that implements BinaryFunctionBase.
*/
type Div struct {
	BinaryFunctionBase
}

/*
The function NewDiv calls NewBinaryFunctionBase to define div
with input operand expressions as input.
*/
func NewDiv(first, second Expression) Function {
	rv := &Div{
		*NewBinaryFunctionBase("div", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitDiv method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Div) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDiv(this)
}

/*
It returns a value type Number.
*/
func (this *Div) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for Binary functions and passes in the
receiver, current item and current context.
*/
func (this *Div) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method evaluates the division for the first and second input 
values to return a value. If the second value type is a number,
convert to a valid Go type. Check for divide by 0. If true return
a Null value. If the first value is a Number, divide the two values
 and return it. If either of the two values are missing return a 
missing value. If not a number and not missing return a NULL value. 
*/
func (this *Div) Apply(context Context, first, second value.Value) (value.Value, error) {
	if second.Type() == value.NUMBER {
		s := second.Actual().(float64)
		if s == 0.0 {
			return value.NULL_VALUE, nil
		}

		if first.Type() == value.NUMBER {
			d := first.Actual().(float64) / s
			return value.NewValue(d), nil
		}
	}

	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	return value.NULL_VALUE, nil
}

/*
The constructor returns a NewDiv with the an operand
cast to a Function as the FunctionConstructor.
*/
func (this *Div) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewDiv(operands[0], operands[1])
	}
}
