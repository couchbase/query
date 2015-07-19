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
	"math"

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

func (this *NNF) VisitLike(expr *expression.Like) (interface{}, error) {
	err := expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	re := expr.Regexp()
	if re == nil {
		return expr, nil
	}

	prefix, complete := re.LiteralPrefix()
	if complete {
		eq := expression.NewEq(expr.First(), expression.NewConstant(prefix))
		return eq, nil
	}

	if prefix == "" {
		return expr, nil
	}

	var and expression.Expression
	le := expression.NewLE(expression.NewConstant(prefix), expr.First())
	last := len(prefix) - 1
	if prefix[last] < math.MaxUint8 {
		bytes := []byte(prefix)
		bytes[last]++
		and = expression.NewAnd(le, expression.NewLT(
			expr.First(),
			expression.NewConstant(string(bytes))))
	} else {
		and = expression.NewAnd(le, expression.NewLT(
			expr.First(),
			expression.EMPTY_ARRAY_EXPR))
	}

	return and, nil
}

/*
Apply Disjunctive Normal Form.

Convert ANDs of ORs to ORs of ANDs. For example:

(A OR B) AND C => (A AND C) OR (B AND C)

Also apply constant folding. Remove any constant terms.
*/
func (this *NNF) VisitAnd(expr *expression.And) (interface{}, error) {
	err := expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	terms := make(expression.Expressions, 0, len(expr.Operands()))

	for _, term := range expr.Operands() {
		val := term.Value()
		if val == nil {
			terms = append(terms, term)
			continue
		}

		if !val.Truth() {
			return expression.FALSE_EXPR, nil
		}
	}

	if len(terms) < len(expr.Operands()) {
		expr = expression.NewAnd(terms...)
	}

	for i, aterm := range expr.Operands() {
		switch aterm := aterm.(type) {
		case *expression.Or:
			na := len(expr.Operands())
			oterms := make(expression.Expressions, len(aterm.Operands()))

			for j, oterm := range aterm.Operands() {
				aterms := make(expression.Expressions, na)
				for ii, atrm := range expr.Operands() {
					if ii == i {
						aterms[ii] = oterm
					} else {
						aterms[ii] = atrm
					}
				}

				oterms[j] = expression.NewAnd(aterms...)
			}

			rv := expression.NewOr(oterms...)
			return rv, rv.MapChildren(this)
		}
	}

	return expr, nil
}

/*
Apply constant folding. Remove any constant terms.
*/
func (this *NNF) VisitOr(expr *expression.Or) (interface{}, error) {
	err := expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	terms := make(expression.Expressions, 0, len(expr.Operands()))

	for _, term := range expr.Operands() {
		val := term.Value()
		if val == nil {
			terms = append(terms, term)
			continue
		}

		if val.Truth() {
			return expression.TRUE_EXPR, nil
		}
	}

	if len(terms) < len(expr.Operands()) {
		expr = expression.NewOr(terms...)
	}

	return expr, nil
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
