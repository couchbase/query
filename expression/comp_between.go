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
Comparison terms allow for comparing two expressions.
For between and not between, we have three expressions,
the input item and the low and high expressions. Type
Between is a struct that implements TernaryFunctionBase.
*/
type Between struct {
	TernaryFunctionBase
}

/*
The function NewBetween calls NewTernaryFunctionBase to
define the between operation with input operands item,
low and high as input to the function.
*/
func NewBetween(item, low, high Expression) Function {
	rv := &Between{
		*NewTernaryFunctionBase("between", item, low, high),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitBetween method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Between) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitBetween(this)
}

/*
It returns a value type BOOLEAN.
*/
func (this *Between) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for Ternary functions and passes in the
receiver, current item and current context.
*/
func (this *Between) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.TernaryEval(this, item, context)
}

/*
This method evaluates the between comparison operation and returns a
value representing if item value is in between low and high. If any of
the input operands are missing, return missing, and if null return null.
For all other types, check if the item is greater in terms of the N1QL
collation order, than the low value and smaller than high value, and return
true. If not return false.
*/
func (this *Between) Apply(context Context, item, low, high value.Value) (value.Value, error) {
	if item.Type() == value.MISSING || low.Type() == value.MISSING || high.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if item.Type() == value.NULL || low.Type() == value.NULL || high.Type() == value.NULL {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(item.Collate(low) >= 0 && item.Collate(high) <= 0), nil
}

/*
The constructor returns a NewEq with the operands
cast to a Function as the FunctionConstructor.
*/
func (this *Between) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewBetween(operands[0], operands[1], operands[2])
	}
}

/*
This function implements the not between operation. It calls
the NewBetween method to return an expression that
is a complement of the NewBetween return type (boolean).
(NewNot represents the Not logical operation)
*/
func NewNotBetween(item, low, high Expression) Expression {
	return NewNot(NewBetween(item, low, high))
}
