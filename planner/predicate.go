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
	"github.com/couchbaselabs/query/expression"
)

type predicate struct {
	test testFunc
}

func newPredicate(test testFunc) *predicate {
	return &predicate{
		test: test,
	}
}

type testFunc func(expression.Expression) (bool, error)

// Arithmetic

func (this *predicate) VisitAdd(expr *expression.Add) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitDiv(expr *expression.Div) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitMod(expr *expression.Mod) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitMult(expr *expression.Mult) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitNeg(expr *expression.Neg) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitSub(expr *expression.Sub) (interface{}, error) {
	return this.test(expr)
}

// Case

func (this *predicate) VisitSearchedCase(expr *expression.SearchedCase) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitSimpleCase(expr *expression.SimpleCase) (interface{}, error) {
	return this.test(expr)
}

// Collection

func (this *predicate) VisitAny(expr *expression.Any) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitArray(expr *expression.Array) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitEvery(expr *expression.Every) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitExists(expr *expression.Exists) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitFirst(expr *expression.First) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitIn(expr *expression.In) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitWithin(expr *expression.Within) (interface{}, error) {
	return this.test(expr)
}

// Comparison

func (this *predicate) VisitBetween(expr *expression.Between) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitEq(expr *expression.Eq) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitLE(expr *expression.LE) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitLike(expr *expression.Like) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitLT(expr *expression.LT) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitIsMissing(expr *expression.IsMissing) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitIsNotMissing(expr *expression.IsNotMissing) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitIsNotNull(expr *expression.IsNotNull) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitIsNull(expr *expression.IsNull) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitIsValued(expr *expression.IsValued) (interface{}, error) {
	return this.test(expr)
}

// Concat
func (this *predicate) VisitConcat(expr *expression.Concat) (interface{}, error) {
	return this.test(expr)
}

// Constant
func (this *predicate) VisitConstant(expr *expression.Constant) (interface{}, error) {
	return this.test(expr)
}

// Identifier
func (this *predicate) VisitIdentifier(expr *expression.Identifier) (interface{}, error) {
	return this.test(expr)
}

// Construction

func (this *predicate) VisitArrayConstruct(expr *expression.ArrayConstruct) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitObjectConstruct(expr *expression.ObjectConstruct) (interface{}, error) {
	return this.test(expr)
}

// Logic

func (this *predicate) VisitAnd(expr *expression.And) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitNot(expr *expression.Not) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitOr(expr *expression.Or) (interface{}, error) {
	return this.test(expr)
}

// Navigation

func (this *predicate) VisitElement(expr *expression.Element) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitField(expr *expression.Field) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitFieldName(expr *expression.FieldName) (interface{}, error) {
	return this.test(expr)
}

func (this *predicate) VisitSlice(expr *expression.Slice) (interface{}, error) {
	return this.test(expr)
}

// Function
func (this *predicate) VisitFunction(expr expression.Function) (interface{}, error) {
	return this.test(expr)
}

// Subquery
func (this *predicate) VisitSubquery(expr expression.Subquery) (interface{}, error) {
	return this.test(expr)
}

// NamedParameter
func (this *predicate) VisitNamedParameter(expr expression.NamedParameter) (interface{}, error) {
	return this.test(expr)
}

// PositionalParameter
func (this *predicate) VisitPositionalParameter(expr expression.PositionalParameter) (interface{}, error) {
	return this.test(expr)
}
