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

var _SARG_FACTORY = &sargFactory{}

type sargFactory struct {
}

// Arithmetic

func (this *sargFactory) VisitAdd(expr *expression.Add) (interface{}, error) {
	return newSargDefault(expr), nil
}

func (this *sargFactory) VisitDiv(expr *expression.Div) (interface{}, error) {
	return newSargDefault(expr), nil
}

func (this *sargFactory) VisitMod(expr *expression.Mod) (interface{}, error) {
	return newSargDefault(expr), nil
}

func (this *sargFactory) VisitMult(expr *expression.Mult) (interface{}, error) {
	return newSargDefault(expr), nil
}

func (this *sargFactory) VisitNeg(expr *expression.Neg) (interface{}, error) {
	return newSargDefault(expr), nil
}

func (this *sargFactory) VisitSub(expr *expression.Sub) (interface{}, error) {
	return newSargDefault(expr), nil
}

// Case

func (this *sargFactory) VisitSearchedCase(expr *expression.SearchedCase) (interface{}, error) {
	return newSargDefault(expr), nil
}

func (this *sargFactory) VisitSimpleCase(expr *expression.SimpleCase) (interface{}, error) {
	return newSargDefault(expr), nil
}

// Collection

func (this *sargFactory) VisitAny(expr *expression.Any) (interface{}, error) {
	return newSargAny(expr), nil
}

func (this *sargFactory) VisitArray(expr *expression.Array) (interface{}, error) {
	return newSargDefault(expr), nil
}

func (this *sargFactory) VisitEvery(expr *expression.Every) (interface{}, error) {
	return newSargDefault(expr), nil
}

func (this *sargFactory) VisitAnyEvery(expr *expression.AnyEvery) (interface{}, error) {
	return newSargAnyEvery(expr), nil
}

func (this *sargFactory) VisitExists(expr *expression.Exists) (interface{}, error) {
	return newSargDefault(expr), nil
}

func (this *sargFactory) VisitFirst(expr *expression.First) (interface{}, error) {
	return newSargDefault(expr), nil
}

func (this *sargFactory) VisitObject(expr *expression.Object) (interface{}, error) {
	return newSargDefault(expr), nil
}

func (this *sargFactory) VisitIn(expr *expression.In) (interface{}, error) {
	return newSargIn(expr), nil
}

func (this *sargFactory) VisitWithin(expr *expression.Within) (interface{}, error) {
	return newSargWithin(expr), nil
}

// Comparison

func (this *sargFactory) VisitBetween(expr *expression.Between) (interface{}, error) {
	return newSargDefault(expr), nil
}

func (this *sargFactory) VisitEq(expr *expression.Eq) (interface{}, error) {
	return newSargEq(expr), nil
}

func (this *sargFactory) VisitLE(expr *expression.LE) (interface{}, error) {
	return newSargLE(expr), nil
}

func (this *sargFactory) VisitLike(expr *expression.Like) (interface{}, error) {
	return newSargLike(expr, expr.Regexp()), nil
}

func (this *sargFactory) VisitLT(expr *expression.LT) (interface{}, error) {
	return newSargLT(expr), nil
}

func (this *sargFactory) VisitIsMissing(expr *expression.IsMissing) (interface{}, error) {
	return newSargDefault(expr), nil
}

func (this *sargFactory) VisitIsNotMissing(expr *expression.IsNotMissing) (interface{}, error) {
	return newSargNotMissing(expr), nil
}

func (this *sargFactory) VisitIsNotNull(expr *expression.IsNotNull) (interface{}, error) {
	return newSargValued(expr), nil
}

func (this *sargFactory) VisitIsNotValued(expr *expression.IsNotValued) (interface{}, error) {
	return newSargDefault(expr), nil
}

func (this *sargFactory) VisitIsNull(expr *expression.IsNull) (interface{}, error) {
	return newSargNull(expr), nil
}

func (this *sargFactory) VisitIsValued(expr *expression.IsValued) (interface{}, error) {
	return newSargValued(expr), nil
}

// Concat
func (this *sargFactory) VisitConcat(expr *expression.Concat) (interface{}, error) {
	return newSargDefault(expr), nil
}

// Constant
func (this *sargFactory) VisitConstant(expr *expression.Constant) (interface{}, error) {
	return newSargDefault(expr), nil
}

// Identifier
func (this *sargFactory) VisitIdentifier(expr *expression.Identifier) (interface{}, error) {
	return newSargDefault(expr), nil
}

// Construction

func (this *sargFactory) VisitArrayConstruct(expr *expression.ArrayConstruct) (interface{}, error) {
	return newSargDefault(expr), nil
}

func (this *sargFactory) VisitObjectConstruct(expr *expression.ObjectConstruct) (interface{}, error) {
	return newSargDefault(expr), nil
}

// Logic

func (this *sargFactory) VisitAnd(expr *expression.And) (interface{}, error) {
	return newSargAnd(expr), nil
}

func (this *sargFactory) VisitNot(expr *expression.Not) (interface{}, error) {
	return newSargDefault(expr), nil
}

func (this *sargFactory) VisitOr(expr *expression.Or) (interface{}, error) {
	return newSargOr(expr), nil
}

// Navigation

func (this *sargFactory) VisitElement(expr *expression.Element) (interface{}, error) {
	return newSargDefault(expr), nil
}

func (this *sargFactory) VisitField(expr *expression.Field) (interface{}, error) {
	return newSargDefault(expr), nil
}

func (this *sargFactory) VisitFieldName(expr *expression.FieldName) (interface{}, error) {
	return newSargDefault(expr), nil
}

func (this *sargFactory) VisitSlice(expr *expression.Slice) (interface{}, error) {
	return newSargDefault(expr), nil
}

// Self
func (this *sargFactory) VisitSelf(expr *expression.Self) (interface{}, error) {
	return newSargDefault(expr), nil
}

// Function
func (this *sargFactory) VisitFunction(expr expression.Function) (interface{}, error) {
	switch expr := expr.(type) {
	case *expression.RegexpLike:
		return newSargLike(expr, expr.Regexp()), nil
	}

	return newSargDefault(expr), nil
}

// Subquery
func (this *sargFactory) VisitSubquery(expr expression.Subquery) (interface{}, error) {
	return newSargDefault(expr), nil
}

// NamedParameter
func (this *sargFactory) VisitNamedParameter(expr expression.NamedParameter) (interface{}, error) {
	return newSargDefault(expr), nil
}

// PositionalParameter
func (this *sargFactory) VisitPositionalParameter(expr expression.PositionalParameter) (interface{}, error) {
	return newSargDefault(expr), nil
}

// Cover
func (this *sargFactory) VisitCover(expr *expression.Cover) (interface{}, error) {
	return expr.Covered().Accept(this)
}

// All
func (this *sargFactory) VisitAll(expr *expression.All) (interface{}, error) {
	return expr.Array().Accept(this)
}
