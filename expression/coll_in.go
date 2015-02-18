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
Represents the Collection expression In. Type In is a
struct that implements BinaryFunctionBase.
*/
type In struct {
	BinaryFunctionBase
}

/*
The function NewIn calls NewBinaryFunctionBase
to define In collection expression with input operand
expressions first and second, as input.
*/
func NewIn(first, second Expression) Function {
	rv := &In{
		*NewBinaryFunctionBase("in", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitIn method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *In) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIn(this)
}

/*
It returns a value type BOOLEAN.
*/
func (this *In) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for Binary functions and passes in the
receiver, current item and current context.
*/
func (this *In) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
IN evaluates to TRUE if the right-hand-side first value is an array
and directly contains the left-hand-side second value. If either
of the input operands are missing, return missing value, and
if the second is not an array return null. Range over the elements of the
array and check if any element is equal to the first value, return true.
For all other cases, return false.
*/
func (this *In) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if second.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	sa := second.Actual().([]interface{})
	for _, s := range sa {
		if first.Equals(value.NewValue(s)) {
			return value.TRUE_VALUE, nil
		}
	}

	return value.FALSE_VALUE, nil
}

/*
The constructor returns a NewIn with the operands
cast to a Function as the FunctionConstructor.
*/
func (this *In) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIn(operands[0], operands[1])
	}
}

/*
This function implements the not in collection operation.
It calls the NewNot over the NewIn to return an expression that
is a complement of the NewIn method boolean return type.
(NewNot represents the Not logical operation)
*/
func NewNotIn(first, second Expression) Expression {
	return NewNot(NewIn(first, second))
}
