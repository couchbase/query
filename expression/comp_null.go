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
Type IsNull is a a struct that implements
UnaryFuncitonBase.
*/
type IsNull struct {
	UnaryFunctionBase
}

/*
The function NewIsNull calls NewUnaryFunctionBase
to define isnull comparison expression with input operand
expression as input.
*/
func NewIsNull(operand Expression) Function {
	rv := &IsNull{
		*NewUnaryFunctionBase("isnull", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitIsNull method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *IsNull) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIsNull(this)
}

/*
It returns a value type BOOLEAN.
*/
func (this *IsNull) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for Unary functions and passes in the
receiver, current item and current context.
*/
func (this *IsNull) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
Evaluates the Is Null comparison operation for expressions.
If the type of input argument is a null value, return true,
if missing return a missing value and by for all other types
return a false value.
*/
func (this *IsNull) Apply(context Context, arg value.Value) (value.Value, error) {
	switch arg.Type() {
	case value.NULL:
		return value.TRUE_VALUE, nil
	case value.MISSING:
		return value.MISSING_VALUE, nil
	default:
		return value.FALSE_VALUE, nil
	}
}

/*
The constructor returns a NewIsNull with the operand
cast to a Function as the FunctionConstructor.
*/
func (this *IsNull) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsNull(operands[0])
	}
}

/*
This function implements the is not null comparison operation.
It calls the NewNot over the NewIsNull to return an expression that
is a complement of its return type (boolean).
(NewNot represents the Not logical operation)
*/
func NewIsNotNull(operand Expression) Expression {
	return NewNot(NewIsNull(operand))
}
