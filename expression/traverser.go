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

func (this *TraverserBase) VisitAdd(expr *Add) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitDiv(expr *Div) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitMod(expr *Mod) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitMult(expr *Mult) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitNeg(expr *Neg) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitSub(expr *Sub) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

// Case

func (this *TraverserBase) VisitSearchedCase(expr *SearchedCase) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitSimpleCase(expr *SimpleCase) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

// Collection

func (this *TraverserBase) VisitExists(expr *Exists) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitIn(expr *In) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitWithin(expr *Within) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitAny(expr *Any) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitEvery(expr *Every) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitAnyEvery(expr *AnyEvery) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitArray(expr *Array) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitFirst(expr *First) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitObject(expr *Object) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

// Comparison

func (this *TraverserBase) VisitBetween(expr *Between) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitEq(expr *Eq) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitLE(expr *LE) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitLike(expr *Like) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitLT(expr *LT) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitIsMissing(expr *IsMissing) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitIsNotMissing(expr *IsNotMissing) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitIsNotNull(expr *IsNotNull) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitIsNotValued(expr *IsNotValued) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitIsNull(expr *IsNull) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitIsValued(expr *IsValued) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

// Concat

func (this *TraverserBase) VisitConcat(expr *Concat) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

// Constant

func (this *TraverserBase) VisitConstant(expr *Constant) (interface{}, error) {
	return expr, nil
}

// Identifier

func (this *TraverserBase) VisitIdentifier(expr *Identifier) (interface{}, error) {
	return expr, nil
}

// Construction

func (this *TraverserBase) VisitArrayConstruct(expr *ArrayConstruct) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitObjectConstruct(expr *ObjectConstruct) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

// Logic

func (this *TraverserBase) VisitAnd(expr *And) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitNot(expr *Not) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitOr(expr *Or) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

// Navigation

func (this *TraverserBase) VisitElement(expr *Element) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitField(expr *Field) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitFieldName(expr *FieldName) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitSlice(expr *Slice) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

// Self

func (this *TraverserBase) VisitSelf(expr *Self) (interface{}, error) {
	return nil, nil
}

// Function

func (this *TraverserBase) VisitFunction(expr Function) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

// Subquery
func (this *TraverserBase) VisitSubquery(expr Subquery) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

// InferUnderParenthesis
func (this *TraverserBase) VisitParenInfer(expr ParenInfer) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

// Parameters

func (this *TraverserBase) VisitNamedParameter(expr NamedParameter) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitPositionalParameter(expr PositionalParameter) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

// Cover
func (this *TraverserBase) VisitCover(expr *Cover) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

// All
func (this *TraverserBase) VisitAll(expr *All) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

// Init
func (this *TraverserBase) SetTraverser(traverser Traverser) {
	if this.traverser == nil {
		this.traverser = traverser
	}
}
