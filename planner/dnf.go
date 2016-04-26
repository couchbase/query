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

type DNF struct {
	expression.MapperBase
	expr         expression.Expression
	dnfTermCount int
}

func NewDNF(expr expression.Expression) *DNF {
	rv := &DNF{
		expr: expr,
	}
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

/*
Convert to Disjunctive Normal Form.

Convert ANDs of ORs to ORs of ANDs. For example:

(A OR B) AND C => (A AND C) OR (B AND C)

Also apply constant folding.
*/
func (this *DNF) VisitAnd(expr *expression.And) (interface{}, error) {
	err := expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	// Flatten nested ANDs
	buffer := make(expression.Expressions, 0, 2*len(expr.Operands()))
	expr = expression.NewAnd(flattenAnd(expr, buffer)...)

	// Constant folding
	for _, term := range expr.Operands() {
		val := term.Value()
		if val != nil && !val.Truth() {
			return expression.FALSE_EXPR, nil
		}
	}

	// DNF
	return this.applyDNF(expr), nil
}

/*
Apply constant folding.
*/
func (this *DNF) VisitOr(expr *expression.Or) (interface{}, error) {
	err := expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	// Flatten nested ORs
	buffer := make(expression.Expressions, 0, 2*len(expr.Operands()))
	expr = expression.NewOr(flattenOr(expr, buffer)...)

	// Constant folding
	for _, term := range expr.Operands() {
		val := term.Value()
		if val != nil && val.Truth() {
			return expression.TRUE_EXPR, nil
		}
	}

	return expr, nil
}

/*
Apply DeMorgan's laws and other transformations.
*/
func (this *DNF) VisitNot(expr *expression.Not) (interface{}, error) {
	err := expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	var exp expression.Expression = expr

	switch operand := expr.Operand().(type) {
	case *expression.Not:
		return operand.Operand(), nil
	case *expression.And:
		operands := make(expression.Expressions, len(operand.Operands()))
		for i, op := range operand.Operands() {
			operands[i] = expression.NewNot(op)
		}

		or := expression.NewOr(operands...)
		return this.VisitOr(or)
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

func flattenOr(or *expression.Or, buffer expression.Expressions) expression.Expressions {
	operands := or.Operands()
	for _, op := range operands {
		switch op := op.(type) {
		case *expression.Or:
			buffer = flattenOr(op, buffer)
		default:
			if len(buffer) == cap(buffer) {
				buffer = growBuffer(buffer)
			}
			buffer = append(buffer, op)
		}
	}

	return buffer
}

func flattenAnd(and *expression.And, buffer expression.Expressions) expression.Expressions {
	operands := and.Operands()
	for _, op := range operands {
		switch op := op.(type) {
		case *expression.And:
			buffer = flattenAnd(op, buffer)
		default:
			if len(buffer) == cap(buffer) {
				buffer = growBuffer(buffer)
			}
			buffer = append(buffer, op)
		}
	}

	return buffer
}

func growBuffer(buffer expression.Expressions) expression.Expressions {
	buf := make(expression.Expressions, len(buffer), 2*len(buffer))
	copy(buf, buffer)
	return buf
}

/*
Bounded DNF, to avoid exponential worst-case.

Internally apply Disjunctive Normal Form.

Convert ANDs of ORs to ORs of ANDs. For example:

(A OR B) AND C => (A AND C) OR (B AND C)
*/
func (this *DNF) applyDNF(expr *expression.And) expression.Expression {
	if this.dnfTermCount >= _MAX_DNF_COMPLEXITY {
		return expr
	}

	complexity := dnfComplexity(expr, _MAX_DNF_COMPLEXITY-this.dnfTermCount)
	if complexity <= 1 || this.dnfTermCount+complexity > _MAX_DNF_COMPLEXITY {
		return expr
	}

	this.dnfTermCount += complexity

	matrix := _EXPRESSIONS_POOL.Get()
	defer _EXPRESSIONS_POOL.Put(matrix)
	matrix = append(matrix, make(expression.Expressions, 0, len(expr.Operands())))

	for _, term := range expr.Operands() {
		switch term := term.(type) {
		case *expression.Or:
			matrix2 := _EXPRESSIONS_POOL.Get()
			defer _EXPRESSIONS_POOL.Put(matrix2)

			orTerms := term.Operands()
			for _, exprs := range matrix {
				for i, orTerm := range orTerms {
					if i < len(orTerms)-1 {
						exprs2 := make(expression.Expressions, len(exprs), cap(exprs))
						copy(exprs2, exprs)
						exprs = exprs2
					}

					switch orTerm := orTerm.(type) {
					case *expression.And:
						// flatten any nested AND
						for _, t := range orTerm.Operands() {
							exprs = append(exprs, t)
						}
					default:
						exprs = append(exprs, orTerm)
					}

					matrix2 = append(matrix2, exprs)
				}
			}

			matrix = matrix2
		default:
			for i, _ := range matrix {
				matrix[i] = append(matrix[i], term)
			}
		}
	}

	terms := make(expression.Expressions, 0, len(matrix))
	for _, exprs := range matrix {
		if len(exprs) == 1 {
			terms = append(terms, exprs[0])
		} else {
			terms = append(terms, expression.NewAnd(exprs...))
		}
	}

	return expression.NewOr(terms...)
}

func dnfComplexity(expr *expression.And, max int) int {
	comp := 1
	for _, op := range expr.Operands() {
		switch op := op.(type) {
		case *expression.Or:
			comp *= len(op.Operands())
			if comp > max {
				break
			}
		}
	}

	return comp
}

const _MAX_DNF_COMPLEXITY = 1024

var _EXPRESSIONS_POOL = expression.NewExpressionsPool(_MAX_DNF_COMPLEXITY)
