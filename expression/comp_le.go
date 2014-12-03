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
Comparison terms allow for comparing two expressions.
This represents the less than equal to comparison
operation. Type LE is a struct that implements
BinaryFunctionBase.
*/
type LE struct {
	BinaryFunctionBase
}

/*
The function NewLE calls NewBinaryFunctionBase
to define less than equal to comparison expression
with input operand expressions first and second,
as input.
*/
func NewLE(first, second Expression) Function {
	rv := &LE{
		*NewBinaryFunctionBase("le", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitLE method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *LE) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitLE(this)
}

/*
It returns a value type BOOLEAN.
*/
func (this *LE) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for Binary functions and passes in the
receiver, current item and current context.
*/
func (this *LE) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method evaluates the less than equal to condition and
returns a value representing if the two operands satisfy the
condition or not. If either of the input operands are
missing, return missing value, and if they are null, then
return null value. For all other types call the Collate
method and check if it is less than equal to 0 for the
two values. If it is, then return true.
*/
func (this *LE) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() == value.NULL || second.Type() == value.NULL {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(first.Collate(second) <= 0), nil
}

/*
The constructor returns a NewLE with the operands
cast to a Function as the FunctionConstructor.
*/
func (this *LE) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewLE(operands[0], operands[1])
	}
}
