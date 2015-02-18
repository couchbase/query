//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"github.com/couchbase/query/expression"
)

type NNF struct {
	expression.MapperBase
}

func NewNNF() *NNF {
	rv := &NNF{}
	rv.SetMapper(rv)
	return rv
}

func (this *NNF) MapBindings() bool { return false }

func (this *NNF) VisitIn(expr *expression.In) (interface{}, error) {
	err := expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	a, ok := expr.Second().(*expression.ArrayConstruct)
	if !ok {
		return expr, nil
	}

	first := expr.First()
	operands := make(expression.Expressions, len(a.Operands()))
	for i, op := range a.Operands() {
		operands[i] = expression.NewEq(first, op)
	}

	return expression.NewOr(operands...), nil
}

func (this *NNF) VisitBetween(expr *expression.Between) (interface{}, error) {
	err := expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expression.NewAnd(expression.NewGE(expr.First(), expr.Second()),
		expression.NewLE(expr.First(), expr.Third())), nil
}

func (this *NNF) VisitNot(expr *expression.Not) (interface{}, error) {
	err := expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	var exp expression.Expression = expr

	switch operand := expr.Operand().(type) {
	case *expression.Not:
		exp = operand.Operand()
	case *expression.And:
		operands := make(expression.Expressions, len(operand.Operands()))
		for i, op := range operand.Operands() {
			operands[i] = expression.NewNot(op)
		}

		exp = expression.NewOr(operands...)
	case *expression.Or:
		operands := make(expression.Expressions, len(operand.Operands()))
		for i, op := range operand.Operands() {
			operands[i] = expression.NewNot(op)
		}

		exp = expression.NewAnd(operands...)
	case *expression.Eq:
		exp = expression.NewOr(expression.NewLT(operand.First(), operand.Second()),
			expression.NewLT(operand.Second(), operand.First()))
	case *expression.LT:
		exp = expression.NewLE(operand.Second(), operand.First())
	case *expression.LE:
		exp = expression.NewLT(operand.Second(), operand.First())
	}

	return exp, exp.MapChildren(this)
}

var _EMPTY_OBJECT_EXPR = expression.NewConstant(map[string]interface{}{})
var _MIN_BINARY_EXPR = expression.NewConstant([]byte{})

func (this *NNF) VisitFunction(expr expression.Function) (interface{}, error) {
	var exp expression.Expression = expr

	switch expr := expr.(type) {
	case *expression.IsBoolean:
		exp = expression.NewLE(expr.Operand(), expression.TRUE_EXPR)
	case *expression.IsNumber:
		exp = expression.NewAnd(
			expression.NewGT(expr.Operand(), expression.TRUE_EXPR),
			expression.NewLT(expr.Operand(), expression.EMPTY_STRING_EXPR))
	case *expression.IsString:
		exp = expression.NewAnd(
			expression.NewGE(expr.Operand(), expression.EMPTY_STRING_EXPR),
			expression.NewLT(expr.Operand(), expression.EMPTY_ARRAY_EXPR))
	case *expression.IsArray:
		exp = expression.NewAnd(
			expression.NewGE(expr.Operand(), expression.EMPTY_ARRAY_EXPR),
			expression.NewLT(expr.Operand(), _EMPTY_OBJECT_EXPR))
	case *expression.IsObject:
		// Not equivalent to IS OBJECT. Includes BINARY values.
		exp = expression.NewGE(expr.Operand(), _EMPTY_OBJECT_EXPR)
	}

	return exp, exp.MapChildren(this)
}
