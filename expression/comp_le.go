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
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For LE, simply list this expression.
*/
func (this *LE) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *LE) Apply(context Context, first, second value.Value) (value.Value, error) {
	cmp := first.Compare(second)
	switch actual := cmp.Actual().(type) {
	case float64:
		return value.NewValue(actual <= 0), nil
	}

	return cmp, nil
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
