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
	expr         expression.Expression
	dnfTermCount int
	like         bool
	doDNF        bool
}

func NewDNF(expr expression.Expression, like bool, doDNF bool) *DNF {
	rv := &DNF{
		expr:  expr,
		like:  like,
		doDNF: doDNF,
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

func (this *DNF) VisitLike(expr *expression.Like) (interface{}, error) {
	return this.visitLike(expr)
}

/*
Convert to Disjunctive Normal Form.

Convert ANDs of ORs to ORs of ANDs. For example:

(A OR B) AND C => (A AND C) OR (B AND C)

Also flatten and apply constant folding.
*/
func (this *DNF) VisitAnd(expr *expression.And) (interface{}, error) {
	// Flatten nested ANDs
	var truth bool
	expr, truth = flattenAnd(expr)
	if !truth {
		return expression.FALSE_EXPR, nil
	}

	err := expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	// Flatten nested ANDs
	expr, _ = flattenAnd(expr)

	switch len(expr.Operands()) {
	case 0:
		return expression.TRUE_EXPR, nil
	case 1:
		return expr.Operands()[0], nil
	default:
		if this.doDNF {
			// DNF
			return this.applyDNF(expr), nil
		} else {
			return expr, nil
		}
	}
}

/*
Flatten and apply constant folding.
*/
func (this *DNF) VisitOr(expr *expression.Or) (interface{}, error) {
	// Flatten nested ORs
	var truth bool
	expr, truth = flattenOr(expr)
	if truth {
		return expression.TRUE_EXPR, nil
	}

	err := expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	// Flatten nested ORs
	expr, _ = flattenOr(expr)

	switch len(expr.Operands()) {
	case 0:
		return expression.FALSE_EXPR, nil
	case 1:
		return expr.Operands()[0], nil
	default:
		return expr, nil
	}
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
	case *expression.IsNull:
		return expression.NewIsNotNull(operand.Operand()), nil
	case *expression.IsMissing:
		return expression.NewIsNotMissing(operand.Operand()), nil
	case *expression.IsValued:
		return expression.NewIsNotValued(operand.Operand()), nil
	case *expression.IsNotNull:
		return expression.NewIsNull(operand.Operand()), nil
	case *expression.IsNotMissing:
		return expression.NewIsMissing(operand.Operand()), nil
	case *expression.IsNotValued:
		return expression.NewIsValued(operand.Operand()), nil
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
	case *expression.In:
		second := operand.Second()
		if acons, ok := second.(*expression.ArrayConstruct); ok &&
			len(acons.Operands()) <= _FULL_SPAN_FANOUT {
			return this.visitNotIn(operand.First(), acons)
		}

		return expr, nil
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

/*
Don't transform subqueries
*/

func (this *DNF) VisitSubquery(expr expression.Subquery) (interface{}, error) {
	return expr, nil
}

var _EMPTY_OBJECT_EXPR = expression.NewConstant(map[string]interface{}{})

func (this *DNF) VisitFunction(expr expression.Function) (interface{}, error) {
	err := expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	var exp expression.Expression

	switch expr := expr.(type) {
	case *expression.RegexpLike:
		return this.visitLike(expr)
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
		exp = expression.NewAnd(
			expression.NewGE(expr.Operand(), _EMPTY_OBJECT_EXPR),
			expr)
		return exp, nil // Avoid infinite recursion
	default:
		return expr, nil // Avoid infinite recursion
	}

	return exp, nil
}

func (this *DNF) visitNotIn(first expression.Expression, second *expression.ArrayConstruct) (
	interface{}, error) {

	neqs := make([]expression.Expression, 0, len(second.Operands()))
	for _, s := range second.Operands() {
		neq := expression.NewNE(first, s)
		neqs = append(neqs, neq)
	}

	and := expression.NewAnd(neqs...)
	return this.VisitAnd(and)
}

func flattenOr(or *expression.Or) (*expression.Or, bool) {
	length, flatten, truth := orLength(or)
	if !flatten || truth {
		return or, truth
	}

	buffer := make(expression.Expressions, 0, length)
	terms := _STRING_EXPRESSION_POOL.Get()
	defer _STRING_EXPRESSION_POOL.Put(terms)
	buffer = orTerms(or, buffer, terms)

	return expression.NewOr(buffer...), false
}

func flattenAnd(and *expression.And) (*expression.And, bool) {
	length, flatten, truth := andLength(and)
	if !flatten || !truth {
		return and, truth
	}

	buffer := make(expression.Expressions, 0, length)
	terms := _STRING_EXPRESSION_POOL.Get()
	defer _STRING_EXPRESSION_POOL.Put(terms)
	buffer = andTerms(and, buffer, terms)

	return expression.NewAnd(buffer...), true
}

func orLength(or *expression.Or) (length int, flatten, truth bool) {
	l := 0
	for _, op := range or.Operands() {
		switch op := op.(type) {
		case *expression.Or:
			l, _, truth = orLength(op)
			if truth {
				return
			}
			length += l
			flatten = true
		default:
			val := op.Value()
			if val != nil {
				if val.Truth() {
					truth = true
					return
				}
			} else {
				length++
			}
		}
	}

	return
}

func andLength(and *expression.And) (length int, flatten, truth bool) {
	truth = true
	l := 0
	for _, op := range and.Operands() {
		switch op := op.(type) {
		case *expression.And:
			l, _, truth = andLength(op)
			if !truth {
				return
			}
			length += l
			flatten = true
		default:
			val := op.Value()
			if val != nil {
				if !val.Truth() {
					truth = false
					return
				}
			} else {
				length++
			}
		}
	}

	return
}

func orTerms(or *expression.Or, buffer expression.Expressions,
	terms map[string]expression.Expression) expression.Expressions {
	for _, op := range or.Operands() {
		switch op := op.(type) {
		case *expression.Or:
			buffer = orTerms(op, buffer, terms)
		default:
			val := op.Value()
			if val == nil || val.Truth() {
				str := op.String()
				if _, found := terms[str]; !found {
					terms[str] = op
					buffer = append(buffer, op)
				}
			}
		}
	}

	return buffer
}

func andTerms(and *expression.And, buffer expression.Expressions,
	terms map[string]expression.Expression) expression.Expressions {
	for _, op := range and.Operands() {
		switch op := op.(type) {
		case *expression.And:
			buffer = andTerms(op, buffer, terms)
		default:
			val := op.Value()
			if val == nil || !val.Truth() {
				str := op.String()
				if _, found := terms[str]; !found {
					terms[str] = op
					buffer = append(buffer, op)
				}
			}
		}
	}

	return buffer
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

	var exprs2 expression.Expressions
	for _, term := range expr.Operands() {
		switch term := term.(type) {
		case *expression.Or:
			matrix2 := _EXPRESSIONS_POOL.Get()
			defer _EXPRESSIONS_POOL.Put(matrix2)

			orTerms := term.Operands()
			for _, exprs := range matrix {
				for i, orTerm := range orTerms {
					if i == len(orTerms)-1 {
						exprs2 = exprs
					} else {
						exprs2 = make(expression.Expressions, len(exprs), cap(exprs))
						copy(exprs2, exprs)
					}

					switch orTerm := orTerm.(type) {
					case *expression.And:
						// flatten any nested AND
						for _, t := range orTerm.Operands() {
							exprs2 = append(exprs2, t)
						}
					default:
						exprs2 = append(exprs2, orTerm)
					}

					matrix2 = append(matrix2, exprs2)
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

func (this *DNF) visitLike(expr expression.LikeFunction) (interface{}, error) {
	err := expr.MapChildren(this)
	if err != nil || !this.like {
		return expr, err
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

	last := len(prefix) - 1
	if last < 0 || prefix[last] >= math.MaxUint8 {
		return expr, nil
	}

	if re.NumSubexp() != 1 || re.String()[len(prefix):] != "(.*)" {
		return expr, nil
	}

	// Now exactSpan = true, so we normalize to comparison
	// operators.

	ge := expression.NewGE(expr.First(), expression.NewConstant(prefix))
	bytes := []byte(prefix)
	bytes[last]++
	lt := expression.NewLT(expr.First(), expression.NewConstant(string(bytes)))
	and := expression.NewAnd(ge, lt)
	return and, nil
}

const _MAX_DNF_COMPLEXITY = 1024

var _EXPRESSIONS_POOL = expression.NewExpressionsPool(_MAX_DNF_COMPLEXITY)
var _STRING_EXPRESSION_POOL = expression.NewStringExpressionPool(_MAX_DNF_COMPLEXITY)
