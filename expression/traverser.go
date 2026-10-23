//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

/*
A Traverser is a Visitor that traverses an Expression and its
descendants. An implementation of Traverser can accumulate
state. e.g. identifying the subqueries or keyspace references within
an expression tree.
*/
type Traverser interface {
	Visitor

	Traverse(expr Expression) error
}

type TraverserBase struct {
	traverser Traverser
}

func (this *TraverserBase) Traverse(expr Expression) (err error) {
	_, err = expr.Accept(this.traverser)
	return
}

func (this *TraverserBase) TraverseList(exprs Expressions) (err error) {
	for _, expr := range exprs {
		err = this.traverser.Traverse(expr)
		if err != nil {
			return
		}
	}

	return
}

// Arithmetic

func (this *TraverserBase) VisitAdd(expr *Add) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitDiv(expr *Div) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitMod(expr *Mod) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitMult(expr *Mult) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitNeg(expr *Neg) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitSub(expr *Sub) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

// Case

func (this *TraverserBase) VisitSearchedCase(expr *SearchedCase) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitSimpleCase(expr *SimpleCase) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

// Collection

func (this *TraverserBase) VisitExists(expr *Exists) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitIn(expr *In) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitWithin(expr *Within) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitAny(expr *Any) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitEvery(expr *Every) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitAnyEvery(expr *AnyEvery) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitArray(expr *Array) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitFirst(expr *First) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitObject(expr *Object) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

// Comparison

func (this *TraverserBase) VisitBetween(expr *Between) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitEq(expr *Eq) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitLE(expr *LE) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitLike(expr *Like) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitLT(expr *LT) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitIsMissing(expr *IsMissing) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitIsNotMissing(expr *IsNotMissing) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitIsNotNull(expr *IsNotNull) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitIsNotValued(expr *IsNotValued) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitIsNull(expr *IsNull) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitIsValued(expr *IsValued) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

// Concat

func (this *TraverserBase) VisitConcat(expr *Concat) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

// Constant

func (this *TraverserBase) VisitConstant(expr *Constant) (any, error) {
	return expr, nil
}

// Identifier

func (this *TraverserBase) VisitIdentifier(expr *Identifier) (any, error) {
	return expr, nil
}

// Construction

func (this *TraverserBase) VisitArrayConstruct(expr *ArrayConstruct) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitObjectConstruct(expr *ObjectConstruct) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

// Logic

func (this *TraverserBase) VisitAnd(expr *And) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitNot(expr *Not) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitOr(expr *Or) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

// Navigation

func (this *TraverserBase) VisitElement(expr *Element) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitField(expr *Field) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitFieldName(expr *FieldName) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitSlice(expr *Slice) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

// Self

func (this *TraverserBase) VisitSelf(expr *Self) (any, error) {
	return nil, nil
}

// Function

func (this *TraverserBase) VisitFunction(expr Function) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

// Subquery
func (this *TraverserBase) VisitSubquery(expr Subquery) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

// InferUnderParenthesis
func (this *TraverserBase) VisitParenInfer(expr ParenInfer) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

// Parameters

func (this *TraverserBase) VisitNamedParameter(expr NamedParameter) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitPositionalParameter(expr PositionalParameter) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

// Cover
func (this *TraverserBase) VisitCover(expr *Cover) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

// All
func (this *TraverserBase) VisitAll(expr *All) (any, error) {
	return nil, this.TraverseList(expr.Children())
}

// Init
func (this *TraverserBase) SetTraverser(traverser Traverser) {
	if this.traverser == nil {
		this.traverser = traverser
	}
}
