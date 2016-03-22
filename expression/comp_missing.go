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
Type IsMissing is a a struct that implements
UnaryFunctionBase.
*/
type IsMissing struct {
	UnaryFunctionBase
}

/*
The function NewIsMissing calls NewUnaryFunctionBase
to define ismissing comparison expression with input operand
expression as input.
*/
func NewIsMissing(operand Expression) Function {
	rv := &IsMissing{
		*NewUnaryFunctionBase("ismissing", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitIsMissing method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *IsMissing) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIsMissing(this)
}

/*
It returns a value type BOOLEAN.
*/
func (this *IsMissing) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for Unary functions and passes in the
receiver, current item and current context.
*/
func (this *IsMissing) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *IsMissing) PropagatesMissing() bool {
	return false
}

func (this *IsMissing) PropagatesNull() bool {
	return false
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For IsMissing, simply list this expression.
*/
func (this *IsMissing) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

/*
Evaluates the Is Missing comparison operation for expressions.
Return true if the input argument value is a missing value,
else return false.
*/
func (this *IsMissing) Apply(context Context, arg value.Value) (value.Value, error) {
	switch arg.Type() {
	case value.MISSING:
		return value.TRUE_VALUE, nil
	default:
		return value.FALSE_VALUE, nil
	}
}

/*
The constructor returns a NewIsMissing with the operands
cast to a Function as the FunctionConstructor.
*/
func (this *IsMissing) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsMissing(operands[0])
	}
}

type IsNotMissing struct {
	UnaryFunctionBase
}

func NewIsNotMissing(operand Expression) Function {
	rv := &IsNotMissing{
		*NewUnaryFunctionBase("isnotmissing", operand),
	}

	rv.expr = rv
	return rv
}

func (this *IsNotMissing) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIsNotMissing(this)
}

func (this *IsNotMissing) Type() value.Type { return value.BOOLEAN }

func (this *IsNotMissing) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *IsNotMissing) PropagatesMissing() bool {
	return false
}

func (this *IsNotMissing) PropagatesNull() bool {
	return false
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For IsNotMissing, simply list this expression.
*/
func (this *IsNotMissing) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *IsNotMissing) Apply(context Context, arg value.Value) (value.Value, error) {
	switch arg.Type() {
	case value.MISSING:
		return value.FALSE_VALUE, nil
	default:
		return value.TRUE_VALUE, nil
	}
}

func (this *IsNotMissing) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsNotMissing(operands[0])
	}
}
