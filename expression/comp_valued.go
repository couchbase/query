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
Type IsValued is a a struct that implements
UnaryFuncitonBase.
*/
type IsValued struct {
	UnaryFunctionBase
}

/*
The function NewIsValued calls NewUnaryFunctionBase
to define isvalued comparison expression with input operand
expression as input.
*/
func NewIsValued(operand Expression) Function {
	rv := &IsValued{
		*NewUnaryFunctionBase("isvalued", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitIsValued method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *IsValued) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIsValued(this)
}

/*
It returns a value type BOOLEAN.
*/
func (this *IsValued) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for Unary functions and passes in the
receiver, current item and current context.
*/
func (this *IsValued) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
Evaluates the Is Valued comparison operation for expressions.
Return true if the input argument value is greater than a null
value, as per N1QL collation order, else return false.
*/
func (this *IsValued) Apply(context Context, arg value.Value) (value.Value, error) {
	return value.NewValue(arg.Type() > value.NULL), nil
}

/*
The constructor returns a NewIsValued with the operand
cast to a Function as the FunctionConstructor.
*/
func (this *IsValued) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsValued(operands[0])
	}
}

type IsNotValued struct {
	UnaryFunctionBase
}

func NewIsNotValued(operand Expression) Function {
	rv := &IsNotValued{
		*NewUnaryFunctionBase("isnotvalued", operand),
	}

	rv.expr = rv
	return rv
}

func (this *IsNotValued) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIsNotValued(this)
}

func (this *IsNotValued) Type() value.Type { return value.BOOLEAN }

func (this *IsNotValued) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *IsNotValued) Apply(context Context, arg value.Value) (value.Value, error) {
	return value.NewValue(arg.Type() <= value.NULL), nil
}

func (this *IsNotValued) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsNotValued(operands[0])
	}
}
