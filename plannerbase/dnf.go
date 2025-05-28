//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plannerbase

import (
	"math"

	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/util"
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

	exp1 := expression.NewGE(expr.First(), expr.Second())
	exp2 := expression.NewLE(expr.First(), expr.Third())
	exp := expression.NewAnd(exp1, exp2)
	exp1.SetExprFlag(expression.EXPR_DERIVED_RANGE1)
	exp2.SetExprFlag(expression.EXPR_DERIVED_RANGE2)
	exp.SetExprFlag(expression.EXPR_DERIVED_RANGE)

	return exp, nil
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
	expr, truth = expression.FlattenAnd(expr)
	if !truth {
		return expression.FALSE_EXPR, nil
	}

	err := expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	// Flatten nested ANDs
	expr, _ = expression.FlattenAnd(expr)

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
	expr, truth = expression.FlattenOr(expr)
	if truth {
		return expression.TRUE_EXPR, nil
	}

	err := expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	// Flatten nested ORs
	expr, _ = expression.FlattenOr(expr)

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
			len(acons.Operands()) <= util.FullSpanFanout(false) {
			return this.visitNotIn(operand.First(), acons)
		}

		return expr, nil
	case *expression.Eq:
		exp = expression.NewOr(expression.NewLT(operand.First(), operand.Second()),
			expression.NewLT(operand.Second(), operand.First()))
		exp.SetExprFlag(expression.EXPR_OR_FROM_NE)
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
		exp1 := expression.NewGT(expr.Operand(), expression.TRUE_EXPR)
		exp2 := expression.NewLT(expr.Operand(), expression.EMPTY_STRING_EXPR)
		exp1.SetExprFlag(expression.EXPR_DERIVED_RANGE1)
		exp2.SetExprFlag(expression.EXPR_DERIVED_RANGE2)
		exp = expression.NewAnd(exp1, exp2)
		exp.SetExprFlag(expression.EXPR_DERIVED_RANGE)
	case *expression.IsString:
		exp1 := expression.NewGE(expr.Operand(), expression.EMPTY_STRING_EXPR)
		exp2 := expression.NewLT(expr.Operand(), expression.EMPTY_ARRAY_EXPR)
		exp1.SetExprFlag(expression.EXPR_DERIVED_RANGE1)
		exp2.SetExprFlag(expression.EXPR_DERIVED_RANGE2)
		exp = expression.NewAnd(exp1, exp2)
		exp.SetExprFlag(expression.EXPR_DERIVED_RANGE)
	case *expression.IsArray:
		exp1 := expression.NewGE(expr.Operand(), expression.EMPTY_ARRAY_EXPR)
		exp2 := expression.NewLT(expr.Operand(), expression.EMPTY_OBJECT_EXPR)
		exp1.SetExprFlag(expression.EXPR_DERIVED_RANGE1)
		exp2.SetExprFlag(expression.EXPR_DERIVED_RANGE2)
		if expr.HasExprFlag(expression.EXPR_UNNEST_ISARRAY) {
			exp1.SetExprFlag(expression.EXPR_UNNEST_ISARRAY)
			exp2.SetExprFlag(expression.EXPR_UNNEST_ISARRAY)
		}
		exp = expression.NewAnd(exp1, exp2)
		exp.SetExprFlag(expression.EXPR_DERIVED_RANGE)
	case *expression.IsObject:
		exp = expression.NewGE(expr.Operand(), expression.EMPTY_OBJECT_EXPR)
		expr.SetExprFlag(expression.EXPR_DERIVED_FROM_ISOBJECT)
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
		expr.SetExprFlag(expression.EXPR_DEFAULT_LIKE)
		return expr, nil
	}

	prefix, complete := re.LiteralPrefix()
	if complete {
		eq := expression.NewEq(expr.First(), expression.NewConstant(prefix))
		return eq, nil
	}

	if prefix == "" {
		expr.SetExprFlag(expression.EXPR_DEFAULT_LIKE)
		return expr, nil
	}

	last := len(prefix) - 1
	if last < 0 || prefix[last] >= math.MaxUint8 {
		expr.SetExprFlag(expression.EXPR_DEFAULT_LIKE)
		return expr, nil
	}

	if re.NumSubexp() != 1 || re.String()[len(prefix):] != "(.*)" {
		expr.SetExprFlag(expression.EXPR_DEFAULT_LIKE)
		return expr, nil
	}

	// Now exactSpan = true, so we normalize to comparison
	// operators.

	ge := expression.NewGE(expr.First(), expression.NewConstant(prefix))
	bytes := []byte(prefix)
	bytes[last]++
	lt := expression.NewLT(expr.First(), expression.NewConstant(string(bytes)))
	ge.SetExprFlag(expression.EXPR_DERIVED_FROM_LIKE)
	lt.SetExprFlag(expression.EXPR_DERIVED_FROM_LIKE)
	and := expression.NewAnd(ge, lt)
	and.SetExprFlag(expression.EXPR_DERIVED_RANGE)
	return and, nil
}

// no need to do DNF transformation for CASE expressions
func (this *DNF) VisitSearchedCase(expr *expression.SearchedCase) (interface{}, error) {
	return expr, nil
}

func (this *DNF) VisitSimpleCase(expr *expression.SimpleCase) (interface{}, error) {
	return expr, nil
}

const _MAX_DNF_COMPLEXITY = 1024

var _EXPRESSIONS_POOL = expression.NewExpressionsPool(_MAX_DNF_COMPLEXITY)
