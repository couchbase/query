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
Comparison terms allow for comparing two expressions. For
equal (= and ==) and not equal (!= and <>) and two forms
are supported to aid in compatibility with other query
languages. Type Eq is a struct that implements
CommutativeBinaryFunctionBase.
*/
type Eq struct {
	CommutativeBinaryFunctionBase
}

/*
The function NewEq calls NewCommutativeBinaryFunctionBase
to define equal comparison expression with input operand
expressions first and second, as input.
*/
func NewEq(first, second Expression) Function {
	rv := &Eq{
		*NewCommutativeBinaryFunctionBase("eq", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitEq method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Eq) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitEq(this)
}

/*
It returns a value type BOOLEAN.
*/
func (this *Eq) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for Binary functions and passes in the
receiver, current item and current context.
*/
func (this *Eq) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method evaluates the equal condition and returns a value
representing if the two operands are equal or not. If either
of the input operands are missing, return missing value, and
if they are null, then return null value. For all other types
call Equals to check if the two values are equal and return
its return value.
*/
func (this *Eq) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() == value.NULL || second.Type() == value.NULL {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(first.Equals(second)), nil
}

/*
The constructor returns a NewEq with the operands
cast to a Function as the FunctionConstructor.
*/
func (this *Eq) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewEq(operands[0], operands[1])
	}
}

/*
This function implements the not equal to comparison operation.
It calls the NewNot over the NewEq to return an expression that
is a complement of the Equal to return type (boolean).
(NewNot represents the Not logical operation)
*/
func NewNE(first, second Expression) Expression {
	return NewNot(NewEq(first, second))
}
