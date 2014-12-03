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
Represents the Collection expression Within. Type Within is a
struct that implements BinaryFunctionBase.
*/
type Within struct {
	BinaryFunctionBase
}

/*
The function NewWithin calls NewBinaryFunctionBase
to define In collection expression with input operand
expressions first and second, as input.
*/
func NewWithin(first, second Expression) Function {
	rv := &Within{
		*NewBinaryFunctionBase("within", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitWithin method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Within) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitWithin(this)
}

/*
It returns a value type BOOLEAN.
*/
func (this *Within) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for Binary functions and passes in the
receiver, current item and current context.
*/
func (this *Within) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
WITHIN evaluates to TRUE if the right-hand-side first value contains
the left-hand-side second value (or name and value) as a child or
descendant (i.e. directly or indirectly). If either
of the input operands are missing, return missing value, and if
the first operand is not an array and the second is not an object
return a null value. Range through the descendants of the second
value object, and check if the first value is equal to each
descendant; return true. For all other cases return false.
*/
func (this *Within) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if second.Type() != value.ARRAY && second.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	desc := second.Descendants(make([]interface{}, 0, 64))
	for _, d := range desc {
		if first.Equals(value.NewValue(d)) {
			return value.TRUE_VALUE, nil
		}
	}

	return value.FALSE_VALUE, nil
}

/*
The constructor returns a NewWithin with the operands
cast to a Function as the FunctionConstructor.
*/
func (this *Within) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewWithin(operands[0], operands[1])
	}
}

/*
This function implements the not Within collection operation.
It calls the NewNot over the NewWithin to return an expression that
is a complement of its boolean return type.
(NewNot represents the Not logical operation)
*/
func NewNotWithin(first, second Expression) Expression {
	return NewNot(NewWithin(first, second))
}
