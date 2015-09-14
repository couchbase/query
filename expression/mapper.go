//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

/*
Mapper is a Visitor that returns an Expression.
*/
type Mapper interface {
	Visitor

	Map(expr Expression) (Expression, error)
}

type MapFunc func(expr Expression) (Expression, error)

type MapperBase struct {
	mapper  Mapper
	mapFunc MapFunc
}

func (this *MapperBase) Map(expr Expression) (Expression, error) {
	exp, err := expr.Accept(this.mapper)
	if err != nil {
		return nil, err
	}

	return exp.(Expression), nil
}

func (this *MapperBase) visit(expr Expression) (interface{}, error) {
	if this.mapFunc != nil {
		return this.mapFunc(expr)
	} else {
		return expr, expr.MapChildren(this.mapper)
	}
}

// Arithmetic

func (this *MapperBase) VisitAdd(expr *Add) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitDiv(expr *Div) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitMod(expr *Mod) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitMult(expr *Mult) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitNeg(expr *Neg) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitSub(expr *Sub) (interface{}, error) {
	return this.visit(expr)
}

// Case

func (this *MapperBase) VisitSearchedCase(expr *SearchedCase) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitSimpleCase(expr *SimpleCase) (interface{}, error) {
	return this.visit(expr)
}

// Collection

func (this *MapperBase) VisitExists(expr *Exists) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitIn(expr *In) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitWithin(expr *Within) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitAny(expr *Any) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitEvery(expr *Every) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitArray(expr *Array) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitFirst(expr *First) (interface{}, error) {
	return this.visit(expr)
}

// Comparison

func (this *MapperBase) VisitBetween(expr *Between) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitEq(expr *Eq) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitLE(expr *LE) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitLike(expr *Like) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitLT(expr *LT) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitIsMissing(expr *IsMissing) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitIsNotMissing(expr *IsNotMissing) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitIsNotNull(expr *IsNotNull) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitIsNotValued(expr *IsNotValued) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitIsNull(expr *IsNull) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitIsValued(expr *IsValued) (interface{}, error) {
	return this.visit(expr)
}

// Concat

func (this *MapperBase) VisitConcat(expr *Concat) (interface{}, error) {
	return this.visit(expr)
}

// Constant

func (this *MapperBase) VisitConstant(expr *Constant) (interface{}, error) {
	return this.visit(expr)
}

// Identifier

func (this *MapperBase) VisitIdentifier(expr *Identifier) (interface{}, error) {
	return this.visit(expr)
}

// Construction

func (this *MapperBase) VisitArrayConstruct(expr *ArrayConstruct) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitObjectConstruct(expr *ObjectConstruct) (interface{}, error) {
	return this.visit(expr)
}

// Logic

func (this *MapperBase) VisitAnd(expr *And) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitNot(expr *Not) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitOr(expr *Or) (interface{}, error) {
	return this.visit(expr)
}

// Navigation

func (this *MapperBase) VisitElement(expr *Element) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitField(expr *Field) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitFieldName(expr *FieldName) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitSlice(expr *Slice) (interface{}, error) {
	return this.visit(expr)
}

// Self

func (this *MapperBase) VisitSelf(expr *Self) (interface{}, error) {
	return this.visit(expr)
}

// Function

func (this *MapperBase) VisitFunction(expr Function) (interface{}, error) {
	return this.visit(expr)
}

// Subquery
func (this *MapperBase) VisitSubquery(expr Subquery) (interface{}, error) {
	return this.visit(expr)
}

// Parameters

func (this *MapperBase) VisitNamedParameter(expr NamedParameter) (interface{}, error) {
	return this.visit(expr)
}

func (this *MapperBase) VisitPositionalParameter(expr PositionalParameter) (interface{}, error) {
	return this.visit(expr)
}

// Cover
func (this *MapperBase) VisitCover(expr *Cover) (interface{}, error) {
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
