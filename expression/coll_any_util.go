//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"fmt"
)

// Given exprs and arrayKey collect (top level) original and replaced binding varaible ANY expressions.
func GatherAny(exprs Expressions, arrayKey *All, replaceOnly bool) (map[Expression]Expression, error) {

	if arrayKey == nil {
		return nil, nil
	}

	rv := &gatherAny{level: 0,
		arrayKey:        arrayKey,
		anys:            make(map[Expression]Expression, 4),
		replaceOnly:     replaceOnly,
		renamedBindings: false,
	}

	for _, expr := range exprs {
		_, err := expr.Accept(rv)
		if err != nil {
			return nil, err
		}
	}
	if len(rv.anys) > 0 {
		return rv.anys, nil
	}
	return nil, nil
}

type gatherAny struct {
	level           int
	arrayKey        Expression
	anys            map[Expression]Expression
	renamedBindings bool
	replaceOnly     bool // different binding variables
}

// Arithmetic

func (this *gatherAny) VisitAdd(expr *Add) (any, error) {
	return this.visit(expr)
}

func (this *gatherAny) VisitDiv(expr *Div) (any, error) {
	return this.visit(expr)
}

func (this *gatherAny) VisitMod(expr *Mod) (any, error) {
	return this.visit(expr)
}

func (this *gatherAny) VisitMult(expr *Mult) (any, error) {
	return this.visit(expr)
}

func (this *gatherAny) VisitNeg(expr *Neg) (any, error) {
	return this.visit(expr)
}

func (this *gatherAny) VisitSub(expr *Sub) (any, error) {
	return this.visit(expr)
}

// Case

func (this *gatherAny) VisitSearchedCase(expr *SearchedCase) (any, error) {
	return this.visit(expr)
}

func (this *gatherAny) VisitSimpleCase(expr *SimpleCase) (any, error) {
	return this.visit(expr)
}

// Collection

func (this *gatherAny) VisitArray(expr *Array) (any, error) {
	return nil, nil
}

func (this *gatherAny) VisitExists(expr *Exists) (any, error) {
	return nil, nil
}

func (this *gatherAny) VisitFirst(expr *First) (any, error) {
	return nil, nil
}

func (this *gatherAny) VisitObject(expr *Object) (any, error) {
	return nil, nil
}

func (this *gatherAny) VisitIn(expr *In) (any, error) {
	return this.visit(expr)
}

func (this *gatherAny) VisitWithin(expr *Within) (any, error) {
	return nil, nil
}

// Comparison

func (this *gatherAny) VisitBetween(expr *Between) (any, error) {
	return this.visit(expr)
}

func (this *gatherAny) VisitEq(expr *Eq) (any, error) {
	return this.visit(expr)
}

func (this *gatherAny) VisitLE(expr *LE) (any, error) {
	return this.visit(expr)
}

func (this *gatherAny) VisitLike(expr *Like) (any, error) {
	return this.visit(expr)
}

func (this *gatherAny) VisitLT(expr *LT) (any, error) {
	return this.visit(expr)
}

func (this *gatherAny) VisitIsMissing(expr *IsMissing) (any, error) {
	return nil, nil
}

func (this *gatherAny) VisitIsNotMissing(expr *IsNotMissing) (any, error) {
	return nil, nil
}

func (this *gatherAny) VisitIsNotNull(expr *IsNotNull) (any, error) {
	return nil, nil
}

func (this *gatherAny) VisitIsNotValued(expr *IsNotValued) (any, error) {
	return nil, nil
}

func (this *gatherAny) VisitIsNull(expr *IsNull) (any, error) {
	return nil, nil
}

func (this *gatherAny) VisitIsValued(expr *IsValued) (any, error) {
	return nil, nil
}

// Concat
func (this *gatherAny) VisitConcat(expr *Concat) (any, error) {
	return nil, nil
}

// Constant
func (this *gatherAny) VisitConstant(expr *Constant) (any, error) {
	return nil, nil
}

// Identifier
func (this *gatherAny) VisitIdentifier(expr *Identifier) (any, error) {
	return nil, nil
}

// Construction

func (this *gatherAny) VisitArrayConstruct(expr *ArrayConstruct) (any, error) {
	return nil, nil
}

func (this *gatherAny) VisitObjectConstruct(expr *ObjectConstruct) (any, error) {
	return nil, nil
}

func (this *gatherAny) VisitNot(expr *Not) (any, error) {
	return this.visit(expr)
}

// Navigation

func (this *gatherAny) VisitElement(expr *Element) (any, error) {
	return nil, nil
}

func (this *gatherAny) VisitField(expr *Field) (any, error) {
	return nil, nil
}

func (this *gatherAny) VisitFieldName(expr *FieldName) (any, error) {
	return nil, nil
}

func (this *gatherAny) VisitSlice(expr *Slice) (any, error) {
	return nil, nil
}

// Self
func (this *gatherAny) VisitSelf(expr *Self) (any, error) {
	return nil, nil
}

// Function
func (this *gatherAny) VisitFunction(expr Function) (any, error) {
	return this.visit(expr)
}

// Subquery
func (this *gatherAny) VisitSubquery(expr Subquery) (any, error) {
	if this.renamedBindings && expr.IsCorrelated() {
		return nil, fmt.Errorf("binding variables differ and correlated subquery")
	}
	return nil, nil
}

// InferUnderParenthesis
func (this *gatherAny) VisitParenInfer(expr ParenInfer) (any, error) {
	return this.visit(expr)
}

// NamedParameter
func (this *gatherAny) VisitNamedParameter(expr NamedParameter) (any, error) {
	return nil, nil
}

// PositionalParameter
func (this *gatherAny) VisitPositionalParameter(expr PositionalParameter) (any, error) {
	return nil, nil
}

// Cover
func (this *gatherAny) VisitCover(expr *Cover) (any, error) {
	return nil, nil
}

// All
func (this *gatherAny) VisitAll(expr *All) (any, error) {
	return nil, nil
}

// For OR, return the intersection over the children
func (this *gatherAny) VisitOr(expr *Or) (any, error) {
	return this.visit(expr)
}

// For AND, return the union over the children
func (this *gatherAny) VisitAnd(expr *And) (any, error) {
	return this.visit(expr)
}

func (this *gatherAny) visit(expr Expression) (any, error) {
	for _, op := range expr.Children() {
		_, err := op.Accept(this)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (this *gatherAny) VisitAnyEvery(expr *AnyEvery) (any, error) {
	return nil, nil
}

func (this *gatherAny) VisitEvery(expr *Every) (any, error) {
	return nil, nil
}

func (this *gatherAny) VisitAny(expr *Any) (any, error) {
	all, ok := this.arrayKey.(*All)
	if !ok {
		return nil, nil
	}

	array, ok := all.Array().(*Array)
	if !ok || !equivalentBindingsWithExpression(expr.Bindings(), array.Bindings(), nil, nil) {
		return nil, nil
	}

	cnflict, renamedBindings, nExpr := renameBindings(expr, all, (this.level == 0))
	any, ok := nExpr.(*Any)
	if cnflict || !ok {
		return nil, fmt.Errorf("Binding variable conflict")
	}
	rv := &gatherAny{level: this.level + 1,
		arrayKey:        array.valueMapping,
		replaceOnly:     this.replaceOnly,
		renamedBindings: renamedBindings,
	}
	if _, err := any.Satisfies().Accept(rv); err != nil {
		return nil, err
	}
	if this.level == 0 && (!this.replaceOnly || !expr.EquivalentTo(any)) {
		this.anys[expr.Copy()] = any
	}
	return nil, nil
}
