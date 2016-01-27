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

var _SUBSET_FACTORY = &subsetFactory{}

type subsetFactory struct {
}

// Arithmetic

func (this *subsetFactory) VisitAdd(expr *expression.Add) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

func (this *subsetFactory) VisitDiv(expr *expression.Div) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

func (this *subsetFactory) VisitMod(expr *expression.Mod) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

func (this *subsetFactory) VisitMult(expr *expression.Mult) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

func (this *subsetFactory) VisitNeg(expr *expression.Neg) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

func (this *subsetFactory) VisitSub(expr *expression.Sub) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

// Case

func (this *subsetFactory) VisitSearchedCase(expr *expression.SearchedCase) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

func (this *subsetFactory) VisitSimpleCase(expr *expression.SimpleCase) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

// Collection

func (this *subsetFactory) VisitAny(expr *expression.Any) (interface{}, error) {
	return newSubsetAny(expr), nil
}

func (this *subsetFactory) VisitArray(expr *expression.Array) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

func (this *subsetFactory) VisitEvery(expr *expression.Every) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

func (this *subsetFactory) VisitExists(expr *expression.Exists) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

func (this *subsetFactory) VisitFirst(expr *expression.First) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

func (this *subsetFactory) VisitIn(expr *expression.In) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

func (this *subsetFactory) VisitWithin(expr *expression.Within) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

// Comparison

func (this *subsetFactory) VisitBetween(expr *expression.Between) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

func (this *subsetFactory) VisitEq(expr *expression.Eq) (interface{}, error) {
	return newSubsetEq(expr), nil
}

func (this *subsetFactory) VisitLE(expr *expression.LE) (interface{}, error) {
	return newSubsetLE(expr), nil
}

func (this *subsetFactory) VisitLike(expr *expression.Like) (interface{}, error) {
	return newSubsetLike(expr, expr.Regexp()), nil
}

func (this *subsetFactory) VisitLT(expr *expression.LT) (interface{}, error) {
	return newSubsetLT(expr), nil
}

func (this *subsetFactory) VisitIsMissing(expr *expression.IsMissing) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

func (this *subsetFactory) VisitIsNotMissing(expr *expression.IsNotMissing) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

func (this *subsetFactory) VisitIsNotNull(expr *expression.IsNotNull) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

func (this *subsetFactory) VisitIsNotValued(expr *expression.IsNotValued) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

func (this *subsetFactory) VisitIsNull(expr *expression.IsNull) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

func (this *subsetFactory) VisitIsValued(expr *expression.IsValued) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

// Concat
func (this *subsetFactory) VisitConcat(expr *expression.Concat) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

// Constant
func (this *subsetFactory) VisitConstant(expr *expression.Constant) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

// Identifier
func (this *subsetFactory) VisitIdentifier(expr *expression.Identifier) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

// Construction

func (this *subsetFactory) VisitArrayConstruct(expr *expression.ArrayConstruct) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

func (this *subsetFactory) VisitObjectConstruct(expr *expression.ObjectConstruct) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

// Logic

func (this *subsetFactory) VisitAnd(expr *expression.And) (interface{}, error) {
	return newSubsetAnd(expr), nil
}

func (this *subsetFactory) VisitNot(expr *expression.Not) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

func (this *subsetFactory) VisitOr(expr *expression.Or) (interface{}, error) {
	return newSubsetOr(expr), nil
}

// Navigation

func (this *subsetFactory) VisitElement(expr *expression.Element) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

func (this *subsetFactory) VisitField(expr *expression.Field) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

func (this *subsetFactory) VisitFieldName(expr *expression.FieldName) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

func (this *subsetFactory) VisitSlice(expr *expression.Slice) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

// Self
func (this *subsetFactory) VisitSelf(expr *expression.Self) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

// Function
func (this *subsetFactory) VisitFunction(expr expression.Function) (interface{}, error) {
	switch expr := expr.(type) {
	case *expression.RegexpLike:
		return newSubsetLike(expr, expr.Regexp()), nil
	}

	return newSubsetDefault(expr), nil
}

// Subquery
func (this *subsetFactory) VisitSubquery(expr expression.Subquery) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

// NamedParameter
func (this *subsetFactory) VisitNamedParameter(expr expression.NamedParameter) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

// PositionalParameter
func (this *subsetFactory) VisitPositionalParameter(expr expression.PositionalParameter) (interface{}, error) {
	return newSubsetDefault(expr), nil
}

// Cover
func (this *subsetFactory) VisitCover(expr *expression.Cover) (interface{}, error) {
	return expr.Covered().Accept(this)
}

// All
func (this *subsetFactory) VisitAll(expr *expression.All) (interface{}, error) {
	return expr.Array().Accept(this)
}
