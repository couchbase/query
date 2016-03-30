//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"github.com/couchbase/query/expression"
)

var _SARGABLE_FACTORY = &sargableFactory{}

type sargableFactory struct {
}

// Arithmetic

func (this *sargableFactory) VisitAdd(expr *expression.Add) (interface{}, error) {
	return newSargableDefault(expr), nil
}

func (this *sargableFactory) VisitDiv(expr *expression.Div) (interface{}, error) {
	return newSargableDefault(expr), nil
}

func (this *sargableFactory) VisitMod(expr *expression.Mod) (interface{}, error) {
	return newSargableDefault(expr), nil
}

func (this *sargableFactory) VisitMult(expr *expression.Mult) (interface{}, error) {
	return newSargableDefault(expr), nil
}

func (this *sargableFactory) VisitNeg(expr *expression.Neg) (interface{}, error) {
	return newSargableDefault(expr), nil
}

func (this *sargableFactory) VisitSub(expr *expression.Sub) (interface{}, error) {
	return newSargableDefault(expr), nil
}

// Case

func (this *sargableFactory) VisitSearchedCase(expr *expression.SearchedCase) (interface{}, error) {
	return newSargableDefault(expr), nil
}

func (this *sargableFactory) VisitSimpleCase(expr *expression.SimpleCase) (interface{}, error) {
	return newSargableDefault(expr), nil
}

// Collection

func (this *sargableFactory) VisitAny(expr *expression.Any) (interface{}, error) {
	return newSargableAny(expr), nil
}

func (this *sargableFactory) VisitArray(expr *expression.Array) (interface{}, error) {
	return newSargableDefault(expr), nil
}

func (this *sargableFactory) VisitEvery(expr *expression.Every) (interface{}, error) {
	return newSargableDefault(expr), nil
}

func (this *sargableFactory) VisitAnyEvery(expr *expression.AnyEvery) (interface{}, error) {
	return newSargableAnyEvery(expr), nil
}

func (this *sargableFactory) VisitExists(expr *expression.Exists) (interface{}, error) {
	return newSargableDefault(expr), nil
}

func (this *sargableFactory) VisitFirst(expr *expression.First) (interface{}, error) {
	return newSargableDefault(expr), nil
}

func (this *sargableFactory) VisitObject(expr *expression.Object) (interface{}, error) {
	return newSargableDefault(expr), nil
}

func (this *sargableFactory) VisitIn(expr *expression.In) (interface{}, error) {
	return newSargableDefault(expr), nil
}

func (this *sargableFactory) VisitWithin(expr *expression.Within) (interface{}, error) {
	return newSargableDefault(expr), nil
}

// Comparison

func (this *sargableFactory) VisitBetween(expr *expression.Between) (interface{}, error) {
	return newSargableDefault(expr), nil
}

func (this *sargableFactory) VisitEq(expr *expression.Eq) (interface{}, error) {
	return newSargableBinary(expr), nil
}

func (this *sargableFactory) VisitLE(expr *expression.LE) (interface{}, error) {
	return newSargableBinary(expr), nil
}

func (this *sargableFactory) VisitLike(expr *expression.Like) (interface{}, error) {
	return newSargableLike(expr, expr.Regexp()), nil
}

func (this *sargableFactory) VisitLT(expr *expression.LT) (interface{}, error) {
	return newSargableBinary(expr), nil
}

func (this *sargableFactory) VisitIsMissing(expr *expression.IsMissing) (interface{}, error) {
	return newSargableDefault(expr), nil
}

func (this *sargableFactory) VisitIsNotMissing(expr *expression.IsNotMissing) (interface{}, error) {
	return newSargableUnary(expr), nil
}

func (this *sargableFactory) VisitIsNotNull(expr *expression.IsNotNull) (interface{}, error) {
	return newSargableUnary(expr), nil
}

func (this *sargableFactory) VisitIsNotValued(expr *expression.IsNotValued) (interface{}, error) {
	return newSargableDefault(expr), nil
}

func (this *sargableFactory) VisitIsNull(expr *expression.IsNull) (interface{}, error) {
	return newSargableUnary(expr), nil
}

func (this *sargableFactory) VisitIsValued(expr *expression.IsValued) (interface{}, error) {
	return newSargableUnary(expr), nil
}

// Concat
func (this *sargableFactory) VisitConcat(expr *expression.Concat) (interface{}, error) {
	return newSargableDefault(expr), nil
}

// Constant
func (this *sargableFactory) VisitConstant(expr *expression.Constant) (interface{}, error) {
	return newSargableDefault(expr), nil
}

// Identifier
func (this *sargableFactory) VisitIdentifier(expr *expression.Identifier) (interface{}, error) {
	return newSargableDefault(expr), nil
}

// Construction

func (this *sargableFactory) VisitArrayConstruct(expr *expression.ArrayConstruct) (interface{}, error) {
	return newSargableDefault(expr), nil
}

func (this *sargableFactory) VisitObjectConstruct(expr *expression.ObjectConstruct) (interface{}, error) {
	return newSargableDefault(expr), nil
}

// Logic

func (this *sargableFactory) VisitAnd(expr *expression.And) (interface{}, error) {
	return newSargableAnd(expr), nil
}

func (this *sargableFactory) VisitNot(expr *expression.Not) (interface{}, error) {
	return newSargableUnary(expr), nil
}

func (this *sargableFactory) VisitOr(expr *expression.Or) (interface{}, error) {
	return newSargableOr(expr), nil
}

// Navigation

func (this *sargableFactory) VisitElement(expr *expression.Element) (interface{}, error) {
	return newSargableDefault(expr), nil
}

func (this *sargableFactory) VisitField(expr *expression.Field) (interface{}, error) {
	return newSargableDefault(expr), nil
}

func (this *sargableFactory) VisitFieldName(expr *expression.FieldName) (interface{}, error) {
	return newSargableDefault(expr), nil
}

func (this *sargableFactory) VisitSlice(expr *expression.Slice) (interface{}, error) {
	return newSargableDefault(expr), nil
}

// Self
func (this *sargableFactory) VisitSelf(expr *expression.Self) (interface{}, error) {
	return newSargableDefault(expr), nil
}

// Function
func (this *sargableFactory) VisitFunction(expr expression.Function) (interface{}, error) {
	switch expr := expr.(type) {
	case *expression.RegexpLike:
		return newSargableLike(expr, expr.Regexp()), nil
	}

	return newSargableDefault(expr), nil
}

// Subquery
func (this *sargableFactory) VisitSubquery(expr expression.Subquery) (interface{}, error) {
	return newSargableDefault(expr), nil
}

// NamedParameter
func (this *sargableFactory) VisitNamedParameter(expr expression.NamedParameter) (interface{}, error) {
	return newSargableDefault(expr), nil
}

// PositionalParameter
func (this *sargableFactory) VisitPositionalParameter(expr expression.PositionalParameter) (interface{}, error) {
	return newSargableDefault(expr), nil
}

// Cover
func (this *sargableFactory) VisitCover(expr *expression.Cover) (interface{}, error) {
	return expr.Covered().Accept(this)
}

// All
func (this *sargableFactory) VisitAll(expr *expression.All) (interface{}, error) {
	return expr.Array().Accept(this)
}
