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

type DNF struct {
	expression.MapperBase
}

func NewDNF() *DNF {
	rv := &DNF{}
	rv.SetMapper(rv)
	return rv
}

func (this *DNF) VisitBetween(expr *expression.Between) (interface{}, error) {
	err := expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expression.NewAnd(expression.NewGE(expr.First(), expr.Second()),
		expression.NewLE(expr.First(), expr.Third())), nil
}

func (this *DNF) VisitLike(expr *expression.Like) (interface{}, error) {
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
func (this *DNF) VisitAnd(expr *expression.And) (interface{}, error) {
	err := expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	// Constant folding
	var terms expression.Expressions
	for _, term := range expr.Operands() {
		val := term.Value()
		if val == nil {
			if terms == nil {
				terms = make(expression.Expressions, 0, len(expr.Operands()))
			}

			terms = append(terms, term)
			continue
		}

		if !val.Truth() {
			return expression.FALSE_EXPR, nil
		}
	}

	if len(terms) == 0 {
		return expression.TRUE_EXPR, nil
	}

	if len(terms) == 1 {
		return terms[0], nil
	}

	if len(terms) < len(expr.Operands()) {
		expr = expression.NewAnd(terms...)
	}

	// DNF
	if dnfComplexity(expr, 16) >= 16 {
		return expr, nil
	} else {
		return applyDNF(expr, 0), nil
	}
}

/*
Bounded DNF, to mitigate combinatorial worst-case.

Internally apply Disjunctive Normal Form.

Convert ANDs of ORs to ORs of ANDs. For example:

(A OR B) AND C => (A AND C) OR (B AND C)
*/
func applyDNF(expr *expression.And, level int) expression.Expression {
	na := len(expr.Operands())
	if na > 4 {
		return expr
	}

	for i, aterm := range expr.Operands() {
		switch aterm := aterm.(type) {
		case *expression.Or:
			no := len(aterm.Operands())
			if no*na > 8 {
				return expr
			}

			oterms := make(expression.Expressions, no)

			for j, oterm := range aterm.Operands() {
				aterms := make(expression.Expressions, na)
				for ii, atrm := range expr.Operands() {
					if ii == i {
						aterms[ii] = oterm
					} else {
						aterms[ii] = atrm
					}
				}

				if level > 2 {
					oterms[j] = expression.NewAnd(aterms...)
				} else {
					oterms[j] = applyDNF(expression.NewAnd(aterms...), level+1)
				}
			}

			rv := expression.NewOr(oterms...)
			return rv
		}
	}

	return expr
}

func dnfComplexity(expr expression.Expression, max int) int {
	comp := 0

	switch expr := expr.(type) {
	case *expression.Or:
		comp = len(expr.Operands())
	}

	if comp < max {
		children := expr.Children()
		for _, child := range children {
			childComp := dnfComplexity(child, max-comp)
			comp += childComp
			if comp >= max {
				break
			}
		}
	}

	return comp
}

/*
Apply constant folding. Remove any constant terms.
*/
func (this *DNF) VisitOr(expr *expression.Or) (interface{}, error) {
	err := expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	// Constant folding
	var terms expression.Expressions
	for _, term := range expr.Operands() {
		val := term.Value()
		if val == nil {
			if terms == nil {
				terms = make(expression.Expressions, 0, len(expr.Operands()))
			}

			terms = append(terms, term)
			continue
		}

		if val.Truth() {
			return expression.TRUE_EXPR, nil
		}
	}

	if len(terms) == 0 {
		return expression.FALSE_EXPR, nil
	}

	if len(terms) == 1 {
		return terms[0], nil
	}

	if len(terms) < len(expr.Operands()) {
		expr = expression.NewOr(terms...)
	}

	return expr, nil
}

func (this *DNF) VisitNot(expr *expression.Not) (interface{}, error) {
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

		and := expression.NewAnd(operands...)
		return this.VisitAnd(and)
	case *expression.Eq:
		exp = expression.NewOr(expression.NewLT(operand.First(), operand.Second()),
			expression.NewLT(operand.Second(), operand.First()))
	case *expression.LT:
		exp = expression.NewLE(operand.Second(), operand.First())
	case *expression.LE:
		exp = expression.NewLT(operand.Second(), operand.First())
	default:
		return expr, nil
	}

	return exp, exp.MapChildren(this)
}

var _EMPTY_OBJECT_EXPR = expression.NewConstant(map[string]interface{}{})
var _MIN_BINARY_EXPR = expression.NewConstant([]byte{})

func (this *DNF) VisitFunction(expr expression.Function) (interface{}, error) {
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
