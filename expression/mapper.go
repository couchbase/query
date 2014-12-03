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
A type Mapper is of type interface that inherits
from Visitor. It has two methods Map that takes
as input an Expression and returns an Expression
and an error. The method MapBindings returns a
boolean.
*/
type Mapper interface {
	Visitor

	Map(expr Expression) (Expression, error)
	MapBindings() bool
}

type MapperBase struct {
	mapper Mapper
}

func (this *MapperBase) Map(expr Expression) (Expression, error) {
	exp, err := expr.Accept(this.mapper)
	if err != nil {
		return nil, err
	}

	return exp.(Expression), nil
}

// Arithmetic

func (this *MapperBase) VisitAdd(expr *Add) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitDiv(expr *Div) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitMod(expr *Mod) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitMult(expr *Mult) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitNeg(expr *Neg) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitSub(expr *Sub) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

// Case

func (this *MapperBase) VisitSearchedCase(expr *SearchedCase) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitSimpleCase(expr *SimpleCase) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

// Collection

func (this *MapperBase) VisitExists(expr *Exists) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitIn(expr *In) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitWithin(expr *Within) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitAny(expr *Any) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitEvery(expr *Every) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitArray(expr *Array) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitFirst(expr *First) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

// Comparison

func (this *MapperBase) VisitBetween(expr *Between) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitEq(expr *Eq) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitLE(expr *LE) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitLike(expr *Like) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitLT(expr *LT) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitIsMissing(expr *IsMissing) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitIsNotMissing(expr *IsNotMissing) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitIsNotNull(expr *IsNotNull) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitIsNull(expr *IsNull) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitIsValued(expr *IsValued) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

// Concat

func (this *MapperBase) VisitConcat(expr *Concat) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

// Constant

func (this *MapperBase) VisitConstant(expr *Constant) (interface{}, error) {
	return expr, nil
}

// Identifier

func (this *MapperBase) VisitIdentifier(expr *Identifier) (interface{}, error) {
	return expr, nil
}

// Construction

func (this *MapperBase) VisitArrayConstruct(expr *ArrayConstruct) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitObjectConstruct(expr *ObjectConstruct) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

// Logic

func (this *MapperBase) VisitAnd(expr *And) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitNot(expr *Not) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitOr(expr *Or) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

// Navigation

func (this *MapperBase) VisitElement(expr *Element) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitField(expr *Field) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitFieldName(expr *FieldName) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitSlice(expr *Slice) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

// Function

func (this *MapperBase) VisitFunction(expr Function) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

// Subquery
func (this *MapperBase) VisitSubquery(expr Subquery) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

// Parameters

func (this *MapperBase) VisitNamedParameter(expr NamedParameter) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

func (this *MapperBase) VisitPositionalParameter(expr PositionalParameter) (interface{}, error) {
	return expr, expr.MapChildren(this.mapper)
}

// Init
func (this *MapperBase) SetMapper(mapper Mapper) {
	if this.mapper == nil {
		this.mapper = mapper
	}
}
