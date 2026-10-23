//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

// Remove an sub-expression (subterm) from an AND expression
func RemoveExpr(expr, removeExpr Expression) (Expression, error) {
	if expr == nil || removeExpr == nil {
		return expr, nil
	}

	remover := newExprRemover(removeExpr)
	newExpr, err := expr.Accept(remover)
	if err != nil {
		return nil, err
	}

	if newExpr == nil {
		return nil, nil
	}
	return newExpr.(Expression), nil
}

type exprRemover struct {
	removeExpr Expression
}

func newExprRemover(removeExpr Expression) *exprRemover {
	if or, ok := removeExpr.(*Or); ok {
		removeExpr, _ = FlattenOr(or)
	}
	return &exprRemover{
		removeExpr: removeExpr,
	}
}

/*
Only remove expression on AND boundary
*/
func (this *exprRemover) VisitAnd(expr *And) (any, error) {
	var sub any
	var err error

	and, _ := FlattenAnd(expr)
	terms := make(Expressions, 0, len(and.Operands()))
	useNew := false
	for _, op := range and.Operands() {
		sub, err = this.visitDefault(op)
		if err != nil {
			return nil, err
		}
		if sub != nil {
			terms = append(terms, sub.(Expression))
		} else {
			useNew = true
		}
	}

	if !useNew {
		// nothing changed
		return expr, nil
	}

	if len(terms) == 0 {
		return nil, nil
	} else if len(terms) == 1 {
		return terms[0], nil
	}

	return NewAnd(terms...), nil
}

func (this *exprRemover) VisitOr(expr *Or) (any, error) {
	or, _ := FlattenOr(expr)
	return this.visitDefault(or)
}

// Arithmetic

func (this *exprRemover) VisitAdd(pred *Add) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitDiv(pred *Div) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitMod(pred *Mod) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitMult(pred *Mult) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitNeg(pred *Neg) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitSub(pred *Sub) (any, error) {
	return this.visitDefault(pred)
}

// Case

func (this *exprRemover) VisitSearchedCase(pred *SearchedCase) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitSimpleCase(pred *SimpleCase) (any, error) {
	return this.visitDefault(pred)
}

// Collection

func (this *exprRemover) VisitAny(pred *Any) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitEvery(pred *Every) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitAnyEvery(pred *AnyEvery) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitArray(pred *Array) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitFirst(pred *First) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitObject(pred *Object) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitExists(pred *Exists) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitIn(pred *In) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitWithin(pred *Within) (any, error) {
	return this.visitDefault(pred)
}

// Comparison

func (this *exprRemover) VisitBetween(pred *Between) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitEq(pred *Eq) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitLE(pred *LE) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitLike(pred *Like) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitLT(pred *LT) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitIsMissing(pred *IsMissing) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitIsNotMissing(pred *IsNotMissing) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitIsNotNull(pred *IsNotNull) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitIsNotValued(pred *IsNotValued) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitIsNull(pred *IsNull) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitIsValued(pred *IsValued) (any, error) {
	return this.visitDefault(pred)
}

// Concat
func (this *exprRemover) VisitConcat(pred *Concat) (any, error) {
	return this.visitDefault(pred)
}

// Constant
func (this *exprRemover) VisitConstant(pred *Constant) (any, error) {
	return this.visitDefault(pred)
}

// Identifier
func (this *exprRemover) VisitIdentifier(pred *Identifier) (any, error) {
	return this.visitDefault(pred)
}

// Construction

func (this *exprRemover) VisitArrayConstruct(pred *ArrayConstruct) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitObjectConstruct(pred *ObjectConstruct) (any, error) {
	return this.visitDefault(pred)
}

// Logic

func (this *exprRemover) VisitNot(pred *Not) (any, error) {
	return this.visitDefault(pred)
}

// Navigation

func (this *exprRemover) VisitElement(pred *Element) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitField(pred *Field) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitFieldName(pred *FieldName) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitSlice(pred *Slice) (any, error) {
	return this.visitDefault(pred)
}

// Self
func (this *exprRemover) VisitSelf(pred *Self) (any, error) {
	return this.visitDefault(pred)
}

// Function
func (this *exprRemover) VisitFunction(pred Function) (any, error) {
	return this.visitDefault(pred)
}

// Subquery
func (this *exprRemover) VisitSubquery(pred Subquery) (any, error) {
	return this.visitDefault(pred)
}

// InferUnderParenthesis
func (this *exprRemover) VisitParenInfer(pred ParenInfer) (any, error) {
	return this.visitDefault(pred)
}

// NamedParameter
func (this *exprRemover) VisitNamedParameter(pred NamedParameter) (any, error) {
	return this.visitDefault(pred)
}

// PositionalParameter
func (this *exprRemover) VisitPositionalParameter(pred PositionalParameter) (any, error) {
	return this.visitDefault(pred)
}

// Cover
func (this *exprRemover) VisitCover(pred *Cover) (any, error) {
	return this.visitDefault(pred)
}

// All
func (this *exprRemover) VisitAll(pred *All) (any, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) visitDefault(expr Expression) (any, error) {
	if expr.EquivalentTo(this.removeExpr) {
		return nil, nil
	}
	return expr, nil
}
