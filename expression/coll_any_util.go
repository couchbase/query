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

func (this *gatherAny) VisitAdd(expr *Add) (interface{}, error) {
	return this.visit(expr)
}

func (this *gatherAny) VisitDiv(expr *Div) (interface{}, error) {
	return this.visit(expr)
}

func (this *gatherAny) VisitMod(expr *Mod) (interface{}, error) {
	return this.visit(expr)
}

func (this *gatherAny) VisitMult(expr *Mult) (interface{}, error) {
	return this.visit(expr)
}

func (this *gatherAny) VisitNeg(expr *Neg) (interface{}, error) {
	return this.visit(expr)
}

func (this *gatherAny) VisitSub(expr *Sub) (interface{}, error) {
	return this.visit(expr)
}

// Case

func (this *gatherAny) VisitSearchedCase(expr *SearchedCase) (interface{}, error) {
	return this.visit(expr)
}

func (this *gatherAny) VisitSimpleCase(expr *SimpleCase) (interface{}, error) {
	return this.visit(expr)
}

// Collection

func (this *gatherAny) VisitArray(expr *Array) (interface{}, error) {
	return nil, nil
}

func (this *gatherAny) VisitExists(expr *Exists) (interface{}, error) {
	return nil, nil
}

func (this *gatherAny) VisitFirst(expr *First) (interface{}, error) {
	return nil, nil
}

func (this *gatherAny) VisitObject(expr *Object) (interface{}, error) {
	return nil, nil
}

func (this *gatherAny) VisitIn(expr *In) (interface{}, error) {
	return this.visit(expr)
}

func (this *gatherAny) VisitWithin(expr *Within) (interface{}, error) {
	return nil, nil
}

// Comparison

func (this *gatherAny) VisitBetween(expr *Between) (interface{}, error) {
	return this.visit(expr)
}

func (this *gatherAny) VisitEq(expr *Eq) (interface{}, error) {
	return this.visit(expr)
}

func (this *gatherAny) VisitLE(expr *LE) (interface{}, error) {
	return this.visit(expr)
}

func (this *gatherAny) VisitLike(expr *Like) (interface{}, error) {
	return this.visit(expr)
}

func (this *gatherAny) VisitLT(expr *LT) (interface{}, error) {
	return this.visit(expr)
}

func (this *gatherAny) VisitIsMissing(expr *IsMissing) (interface{}, error) {
	return nil, nil
}

func (this *gatherAny) VisitIsNotMissing(expr *IsNotMissing) (interface{}, error) {
	return nil, nil
}

func (this *gatherAny) VisitIsNotNull(expr *IsNotNull) (interface{}, error) {
	return nil, nil
}

func (this *gatherAny) VisitIsNotValued(expr *IsNotValued) (interface{}, error) {
	return nil, nil
}

func (this *gatherAny) VisitIsNull(expr *IsNull) (interface{}, error) {
	return nil, nil
}

func (this *gatherAny) VisitIsValued(expr *IsValued) (interface{}, error) {
	return nil, nil
}

// Concat
func (this *gatherAny) VisitConcat(expr *Concat) (interface{}, error) {
	return nil, nil
}

// Constant
func (this *gatherAny) VisitConstant(expr *Constant) (interface{}, error) {
	return nil, nil
}

// Identifier
func (this *gatherAny) VisitIdentifier(expr *Identifier) (interface{}, error) {
	return nil, nil
}

// Construction

func (this *gatherAny) VisitArrayConstruct(expr *ArrayConstruct) (interface{}, error) {
	return nil, nil
}

func (this *gatherAny) VisitObjectConstruct(expr *ObjectConstruct) (interface{}, error) {
	return nil, nil
}

func (this *gatherAny) VisitNot(expr *Not) (interface{}, error) {
	return this.visit(expr)
}

// Navigation

func (this *gatherAny) VisitElement(expr *Element) (interface{}, error) {
	return nil, nil
}

func (this *gatherAny) VisitField(expr *Field) (interface{}, error) {
	return nil, nil
}

func (this *gatherAny) VisitFieldName(expr *FieldName) (interface{}, error) {
	return nil, nil
}

func (this *gatherAny) VisitSlice(expr *Slice) (interface{}, error) {
	return nil, nil
}

// Self
func (this *gatherAny) VisitSelf(expr *Self) (interface{}, error) {
	return nil, nil
}

// Function
func (this *gatherAny) VisitFunction(expr Function) (interface{}, error) {
	return this.visit(expr)
}

// Subquery
func (this *gatherAny) VisitSubquery(expr Subquery) (interface{}, error) {
	if this.renamedBindings && expr.IsCorrelated() {
		return nil, fmt.Errorf("binding variables differ and correlated subquery")
	}
	return nil, nil
}

// InferUnderParenthesis
func (this *gatherAny) VisitParenInfer(expr ParenInfer) (interface{}, error) {
	return this.visit(expr)
}

// NamedParameter
func (this *gatherAny) VisitNamedParameter(expr NamedParameter) (interface{}, error) {
	return nil, nil
}

// PositionalParameter
func (this *gatherAny) VisitPositionalParameter(expr PositionalParameter) (interface{}, error) {
	return nil, nil
}

// Cover
func (this *gatherAny) VisitCover(expr *Cover) (interface{}, error) {
	return nil, nil
}

// All
func (this *gatherAny) VisitAll(expr *All) (interface{}, error) {
	return nil, nil
}

// For OR, return the intersection over the children
func (this *gatherAny) VisitOr(expr *Or) (interface{}, error) {
	return this.visit(expr)
}

// For AND, return the union over the children
func (this *gatherAny) VisitAnd(expr *And) (interface{}, error) {
	return this.visit(expr)
}

func (this *gatherAny) visit(expr Expression) (interface{}, error) {
	for _, op := range expr.Children() {
		_, err := op.Accept(this)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (this *gatherAny) VisitAnyEvery(expr *AnyEvery) (interface{}, error) {
	return nil, nil
}

func (this *gatherAny) VisitEvery(expr *Every) (interface{}, error) {
	return nil, nil
}

func (this *gatherAny) VisitAny(expr *Any) (interface{}, error) {
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
