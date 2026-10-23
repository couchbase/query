//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

/*
Mapper is a Visitor that returns an Expression.
*/
type Mapper interface {
	Visitor

	Map(expr Expression) (Expression, error)
	SkipSubq() bool
}

type MapFunc func(expr Expression) (Expression, error)

type MapperBase struct {
	mapper   Mapper
	mapFunc  MapFunc
	skipSubq bool
}

func (this *MapperBase) Map(expr Expression) (Expression, error) {
	exp, err := expr.Accept(this.mapper)
	if err != nil {
		return nil, err
	}

	return exp.(Expression), nil
}

func (this *MapperBase) visit(expr Expression) (any, error) {
	if this.mapFunc != nil {
		return this.mapFunc(expr)
	} else {
		return expr, expr.MapChildren(this.mapper)
	}
}

// Arithmetic

func (this *MapperBase) VisitAdd(expr *Add) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitDiv(expr *Div) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitMod(expr *Mod) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitMult(expr *Mult) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitNeg(expr *Neg) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitSub(expr *Sub) (any, error) {
	return this.visit(expr)
}

// Case

func (this *MapperBase) VisitSearchedCase(expr *SearchedCase) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitSimpleCase(expr *SimpleCase) (any, error) {
	return this.visit(expr)
}

// Collection

func (this *MapperBase) VisitExists(expr *Exists) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitIn(expr *In) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitWithin(expr *Within) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitAny(expr *Any) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitEvery(expr *Every) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitAnyEvery(expr *AnyEvery) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitArray(expr *Array) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitFirst(expr *First) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitObject(expr *Object) (any, error) {
	return this.visit(expr)
}

// Comparison

func (this *MapperBase) VisitBetween(expr *Between) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitEq(expr *Eq) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitLE(expr *LE) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitLike(expr *Like) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitLT(expr *LT) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitIsMissing(expr *IsMissing) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitIsNotMissing(expr *IsNotMissing) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitIsNotNull(expr *IsNotNull) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitIsNotValued(expr *IsNotValued) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitIsNull(expr *IsNull) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitIsValued(expr *IsValued) (any, error) {
	return this.visit(expr)
}

// Concat

func (this *MapperBase) VisitConcat(expr *Concat) (any, error) {
	return this.visit(expr)
}

// Constant

func (this *MapperBase) VisitConstant(expr *Constant) (any, error) {
	return this.visit(expr)
}

// Identifier

func (this *MapperBase) VisitIdentifier(expr *Identifier) (any, error) {
	return this.visit(expr)
}

// Construction

func (this *MapperBase) VisitArrayConstruct(expr *ArrayConstruct) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitObjectConstruct(expr *ObjectConstruct) (any, error) {
	return this.visit(expr)
}

// Logic

func (this *MapperBase) VisitAnd(expr *And) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitNot(expr *Not) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitOr(expr *Or) (any, error) {
	return this.visit(expr)
}

// Navigation

func (this *MapperBase) VisitElement(expr *Element) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitField(expr *Field) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitFieldName(expr *FieldName) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitSlice(expr *Slice) (any, error) {
	return this.visit(expr)
}

// Self

func (this *MapperBase) VisitSelf(expr *Self) (any, error) {
	return this.visit(expr)
}

// Function

func (this *MapperBase) VisitFunction(expr Function) (any, error) {
	return this.visit(expr)
}

// Subquery
func (this *MapperBase) VisitSubquery(expr Subquery) (any, error) {
	return this.visit(expr)
}

// InferUnderParenthesis
func (this *MapperBase) VisitParenInfer(expr ParenInfer) (any, error) {
	return this.visit(expr)
}

// Parameters

func (this *MapperBase) VisitNamedParameter(expr NamedParameter) (any, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitPositionalParameter(expr PositionalParameter) (any, error) {
	return this.visit(expr)
}

// Cover
func (this *MapperBase) VisitCover(expr *Cover) (any, error) {
	return this.visit(expr)
}

// All
func (this *MapperBase) VisitAll(expr *All) (any, error) {
	return this.visit(expr)
}

// Init
func (this *MapperBase) SetMapper(mapper Mapper) {
	if this.mapper == nil {
		this.mapper = mapper
	}
}

func (this *MapperBase) SetMapFunc(f MapFunc) {
	if this.mapFunc == nil {
		this.mapFunc = f
	}
}

func (this *MapperBase) SkipSubq() bool {
	return this.skipSubq
}

func (this *MapperBase) SetSkipSubq() {
	this.skipSubq = true
}

func (this *MapperBase) UnsetSkipSubq() {
	this.skipSubq = false
}
