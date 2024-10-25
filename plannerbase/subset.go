//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plannerbase

import (
	"github.com/couchbase/query/expression"
)

func SubsetOf(expr1, expr2 expression.Expression) bool {
	v2 := expr2.Value()
	if v2 != nil {
		return v2.Truth()
	}

	if expr1.EquivalentTo(expr2) {
		return true
	}

	if and, ok := expr2.(*expression.And); ok {
		expr2, _ = expression.FlattenAndNoDedup(and)
	} else if or, ok := expr2.(*expression.Or); ok {
		expr2, _ = expression.FlattenOrNoDedup(or)
	}

	s := &subset{expr2}
	result, _ := expr1.Accept(s)
	return result.(bool)
}

type subset struct {
	expr2 expression.Expression
}

// Arithmetic

func (this *subset) VisitAdd(expr *expression.Add) (interface{}, error) {
	return this.visitDefault(expr)
}

func (this *subset) VisitDiv(expr *expression.Div) (interface{}, error) {
	return this.visitDefault(expr)
}

func (this *subset) VisitMod(expr *expression.Mod) (interface{}, error) {
	return this.visitDefault(expr)
}

func (this *subset) VisitMult(expr *expression.Mult) (interface{}, error) {
	return this.visitDefault(expr)
}

func (this *subset) VisitNeg(expr *expression.Neg) (interface{}, error) {
	return this.visitDefault(expr)
}

func (this *subset) VisitSub(expr *expression.Sub) (interface{}, error) {
	return this.visitDefault(expr)
}

// Case

func (this *subset) VisitSearchedCase(expr *expression.SearchedCase) (interface{}, error) {
	return this.visitDefault(expr)
}

func (this *subset) VisitSimpleCase(expr *expression.SimpleCase) (interface{}, error) {
	return this.visitDefault(expr)
}

// Collection

func (this *subset) VisitArray(expr *expression.Array) (interface{}, error) {
	return this.visitDefault(expr)
}

func (this *subset) VisitExists(expr *expression.Exists) (interface{}, error) {
	return this.visitDefault(expr)
}

func (this *subset) VisitFirst(expr *expression.First) (interface{}, error) {
	return this.visitDefault(expr)
}

func (this *subset) VisitObject(expr *expression.Object) (interface{}, error) {
	return this.visitDefault(expr)
}

// Comparison

func (this *subset) VisitBetween(expr *expression.Between) (interface{}, error) {
	return this.visitDefault(expr)
}

func (this *subset) VisitLike(expr *expression.Like) (interface{}, error) {
	return this.visitLike(expr)
}

func (this *subset) VisitIsMissing(expr *expression.IsMissing) (interface{}, error) {
	return this.visitDefault(expr)
}

func (this *subset) VisitIsNotMissing(expr *expression.IsNotMissing) (interface{}, error) {
	return this.visitDefault(expr)
}

func (this *subset) VisitIsNotNull(expr *expression.IsNotNull) (interface{}, error) {
	return this.visitDefault(expr)
}

func (this *subset) VisitIsNotValued(expr *expression.IsNotValued) (interface{}, error) {
	return this.visitDefault(expr)
}

func (this *subset) VisitIsNull(expr *expression.IsNull) (interface{}, error) {
	return this.visitDefault(expr)
}

func (this *subset) VisitIsValued(expr *expression.IsValued) (interface{}, error) {
	return this.visitDefault(expr)
}

// Concat
func (this *subset) VisitConcat(expr *expression.Concat) (interface{}, error) {
	return this.visitDefault(expr)
}

// Constant
func (this *subset) VisitConstant(expr *expression.Constant) (interface{}, error) {
	return this.visitDefault(expr)
}

// Identifier
func (this *subset) VisitIdentifier(expr *expression.Identifier) (interface{}, error) {
	return this.visitDefault(expr)
}

// Construction

func (this *subset) VisitArrayConstruct(expr *expression.ArrayConstruct) (interface{}, error) {
	return this.visitDefault(expr)
}

func (this *subset) VisitObjectConstruct(expr *expression.ObjectConstruct) (interface{}, error) {
	return this.visitDefault(expr)
}

// Logic

func (this *subset) VisitNot(expr *expression.Not) (interface{}, error) {
	return this.visitDefault(expr)
}

// Navigation

func (this *subset) VisitElement(expr *expression.Element) (interface{}, error) {
	return this.visitDefault(expr)
}

func (this *subset) VisitField(expr *expression.Field) (interface{}, error) {
	return this.visitDefault(expr)
}

func (this *subset) VisitFieldName(expr *expression.FieldName) (interface{}, error) {
	return this.visitDefault(expr)
}

func (this *subset) VisitSlice(expr *expression.Slice) (interface{}, error) {
	return this.visitDefault(expr)
}

// Self
func (this *subset) VisitSelf(expr *expression.Self) (interface{}, error) {
	return this.visitDefault(expr)
}

// Function
func (this *subset) VisitFunction(expr expression.Function) (interface{}, error) {
	switch expr := expr.(type) {
	case *expression.RegexpLike:
		return this.visitLike(expr)
	}

	return this.visitDefault(expr)
}

// Subquery
func (this *subset) VisitSubquery(expr expression.Subquery) (interface{}, error) {
	return this.visitDefault(expr)
}

// InferUnderParenthesis
func (this *subset) VisitParenInfer(expr expression.ParenInfer) (interface{}, error) {
	return this.visitDefault(expr)
}

// NamedParameter
func (this *subset) VisitNamedParameter(expr expression.NamedParameter) (interface{}, error) {
	return this.visitDefault(expr)
}

// PositionalParameter
func (this *subset) VisitPositionalParameter(expr expression.PositionalParameter) (interface{}, error) {
	return this.visitDefault(expr)
}

// Cover
func (this *subset) VisitCover(expr *expression.Cover) (interface{}, error) {
	return expr.Covered().Accept(this)
}

// All
func (this *subset) VisitAll(expr *expression.All) (interface{}, error) {
	return expr.Array().Accept(this)
}

func (this *subset) visitDefault(expr expression.Expression) (bool, error) {
	expr2 := this.expr2
	value2 := expr2.Value()
	if value2 != nil {
		return value2.Truth(), nil
	}

	if expr.EquivalentTo(expr2) {
		return true, nil
	}

	switch expr2 := expr2.(type) {
	case *expression.And:
		for _, op := range expr2.Operands() {
			if !SubsetOf(expr, op) {
				return false, nil
			}
		}

		return true, nil
	case *expression.Or:
		for _, op := range expr2.Operands() {
			if SubsetOf(expr, op) {
				return true, nil
			}
		}

		return false, nil
	case *expression.IsNotMissing:
		return expr.PropagatesMissing() &&
			expr.DependsOn(expr2.Operand()), nil
	case *expression.IsNotNull:
		return expr.PropagatesNull() &&
			expr.DependsOn(expr2.Operand()), nil
	case *expression.IsValued:
		return expr.PropagatesNull() &&
			expr.DependsOn(expr2.Operand()), nil
	}

	return false, nil
}
