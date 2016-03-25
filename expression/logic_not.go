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
Logical terms allow for combining other expressions using boolean logic.
Standard NOT operators are supported. Type Not is a struct that
implements UnaryFunctionBase.
*/
type Not struct {
	UnaryFunctionBase
}

/*
The function NewNot calls NewUnaryFunctionBase to define Not
with input operand expression as input.
*/
func NewNot(operand Expression) Function {
	rv := &Not{
		*NewUnaryFunctionBase("not", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitNot method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Not) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitNot(this)
}

/*
It returns a value type Boolean.
*/
func (this *Not) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *Not) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For NOT, simply list this expression.
*/
func (this *Not) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

/*
Disallow pushing operation to index scan
*/

func (this *Not) IndexPushDownAllowed() bool {
	return false
}

func (this *Not) Apply(context Context, arg value.Value) (value.Value, error) {
	switch arg.Type() {
	case value.MISSING, value.NULL:
		return arg, nil
	default:
		if arg.Truth() {
			return value.FALSE_VALUE, nil
		} else {
			return value.TRUE_VALUE, nil
		}
	}
}

/*
The constructor returns a NewNot by casting the receiver to a
Function as the FunctionConstructor.
*/
func (this *Not) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewNot(operands[0])
	}
}
