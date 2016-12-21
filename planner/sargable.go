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

func SargableFor(pred expression.Expression, keys expression.Expressions) int {
	n := len(keys)

	i := 0
	for ; i < n; i++ {
		// Terminate on statically-valued expression
		if keys[i].Value() != nil {
			return i
		}

		s := &sargable{keys[i]}

		r, err := pred.Accept(s)
		if err != nil || !r.(bool) {
			return i
		}
	}

	return i
}

type sargable struct {
	key expression.Expression
}

// Arithmetic

func (this *sargable) VisitAdd(pred *expression.Add) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitDiv(pred *expression.Div) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitMod(pred *expression.Mod) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitMult(pred *expression.Mult) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitNeg(pred *expression.Neg) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitSub(pred *expression.Sub) (interface{}, error) {
	return this.visitDefault(pred)
}

// Case

func (this *sargable) VisitSearchedCase(pred *expression.SearchedCase) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitSimpleCase(pred *expression.SimpleCase) (interface{}, error) {
	return this.visitDefault(pred)
}

// Collection

func (this *sargable) VisitArray(pred *expression.Array) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitEvery(pred *expression.Every) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitExists(pred *expression.Exists) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitFirst(pred *expression.First) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitObject(pred *expression.Object) (interface{}, error) {
	return this.visitDefault(pred)
}

// Comparison

func (this *sargable) VisitBetween(pred *expression.Between) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitEq(pred *expression.Eq) (interface{}, error) {
	return this.visitBinary(pred)
}

func (this *sargable) VisitLE(pred *expression.LE) (interface{}, error) {
	return this.visitBinary(pred)
}

func (this *sargable) VisitLike(pred *expression.Like) (interface{}, error) {
	return this.visitLike(pred)
}

func (this *sargable) VisitLT(pred *expression.LT) (interface{}, error) {
	return this.visitBinary(pred)
}

func (this *sargable) VisitIsMissing(pred *expression.IsMissing) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitIsNotMissing(pred *expression.IsNotMissing) (interface{}, error) {
	return this.visitUnary(pred)
}

func (this *sargable) VisitIsNotNull(pred *expression.IsNotNull) (interface{}, error) {
	return this.visitUnary(pred)
}

func (this *sargable) VisitIsNotValued(pred *expression.IsNotValued) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitIsNull(pred *expression.IsNull) (interface{}, error) {
	return this.visitUnary(pred)
}

func (this *sargable) VisitIsValued(pred *expression.IsValued) (interface{}, error) {
	return this.visitUnary(pred)
}

// Concat
func (this *sargable) VisitConcat(pred *expression.Concat) (interface{}, error) {
	return this.visitDefault(pred)
}

// Constant
func (this *sargable) VisitConstant(pred *expression.Constant) (interface{}, error) {
	return this.visitDefault(pred)
}

// Identifier
func (this *sargable) VisitIdentifier(pred *expression.Identifier) (interface{}, error) {
	return this.visitDefault(pred)
}

// Construction

func (this *sargable) VisitArrayConstruct(pred *expression.ArrayConstruct) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitObjectConstruct(pred *expression.ObjectConstruct) (interface{}, error) {
	return this.visitDefault(pred)
}

// Logic

func (this *sargable) VisitNot(pred *expression.Not) (interface{}, error) {
	return this.visitUnary(pred)
}

// Navigation

func (this *sargable) VisitElement(pred *expression.Element) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitField(pred *expression.Field) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitFieldName(pred *expression.FieldName) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitSlice(pred *expression.Slice) (interface{}, error) {
	return this.visitDefault(pred)
}

// Self
func (this *sargable) VisitSelf(pred *expression.Self) (interface{}, error) {
	return this.visitDefault(pred)
}

// Function
func (this *sargable) VisitFunction(pred expression.Function) (interface{}, error) {
	switch pred := pred.(type) {
	case *expression.RegexpLike:
		return this.visitLike(pred)
	}

	return this.visitDefault(pred)
}

// Subquery
func (this *sargable) VisitSubquery(pred expression.Subquery) (interface{}, error) {
	return this.visitDefault(pred)
}

// NamedParameter
func (this *sargable) VisitNamedParameter(pred expression.NamedParameter) (interface{}, error) {
	return this.visitDefault(pred)
}

// PositionalParameter
func (this *sargable) VisitPositionalParameter(pred expression.PositionalParameter) (interface{}, error) {
	return this.visitDefault(pred)
}

// Cover
func (this *sargable) VisitCover(pred *expression.Cover) (interface{}, error) {
	return pred.Covered().Accept(this)
}

// All
func (this *sargable) VisitAll(pred *expression.All) (interface{}, error) {
	return pred.Array().Accept(this)
}

func (this *sargable) visitDefault(pred expression.Expression) (bool, error) {
	return this.defaultSargable(pred), nil
}

func (this *sargable) defaultSargable(pred expression.Expression) bool {
	return SubsetOf(pred, this.key) ||
		((pred.PropagatesMissing() || pred.PropagatesNull()) &&
			pred.DependsOn(this.key))
}
