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
Represents the Collection expression Exists. Type Exists
is a struct that implements UnaryFunctionBase.
*/
type Exists struct {
	UnaryFunctionBase
}

/*
The function NewExists calls NewUnaryFunctionBase
to define exists collection expression with input operand
expression as input.
*/
func NewExists(operand Expression) *Exists {
	rv := &Exists{
		*NewUnaryFunctionBase("exists", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitExists method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Exists) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExists(this)
}

/*
It returns a value type BOOLEAN.
*/
func (this *Exists) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for Unary functions and passes in the
receiver, current item and current context.
*/
func (this *Exists) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method returns true if he value is an array and contains at least one element.
This is done by checking the length of the array. If the type of input value
is missing then return a missing value, and for all other types return null.
*/
func (this *Exists) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.ARRAY {
		a := arg.Actual().([]interface{})
		return value.NewValue(len(a) > 0), nil
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else {
		return value.NULL_VALUE, nil
	}
}

/*
The constructor returns a NewExists with the operand
cast to a Function as the FunctionConstructor.
*/
func (this *Exists) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewExists(operands[0])
	}
}
