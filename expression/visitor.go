//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

type Visitor interface {
	// Arithmetic
	VisitAdd(expr *Add) (interface{}, error)
	VisitDiv(expr *Div) (interface{}, error)
	VisitMod(expr *Mod) (interface{}, error)
	VisitMult(expr *Mult) (interface{}, error)
	VisitNeg(expr *Neg) (interface{}, error)
	VisitSub(expr *Sub) (interface{}, error)

	// Case
	VisitSearchedCase(expr *SearchedCase) (interface{}, error)
	VisitSimpleCase(expr *SimpleCase) (interface{}, error)

	// Collection
	VisitAny(expr *Any) (interface{}, error)
	VisitArray(expr *Array) (interface{}, error)
	VisitEvery(expr *Every) (interface{}, error)
	VisitExists(expr *Exists) (interface{}, error)
	VisitFirst(expr *First) (interface{}, error)
	VisitIn(expr *In) (interface{}, error)
	VisitWithin(expr *Within) (interface{}, error)

	// Comparison
	VisitBetween(expr *Between) (interface{}, error)
	VisitEq(expr *Eq) (interface{}, error)
	VisitLE(expr *LE) (interface{}, error)
	VisitLike(expr *Like) (interface{}, error)
	VisitLT(expr *LT) (interface{}, error)
	VisitIsMissing(expr *IsMissing) (interface{}, error)
	VisitIsNull(expr *IsNull) (interface{}, error)
	VisitIsValued(expr *IsValued) (interface{}, error)

	// Concat
	VisitConcat(expr *Concat) (interface{}, error)

	// Constant
	VisitConstant(expr *Constant) (interface{}, error)

	// Identifier
	VisitIdentifier(expr *Identifier) (interface{}, error)

	// Construction
	VisitArrayConstruct(expr *ArrayConstruct) (interface{}, error)
	VisitObjectConstruct(expr *ObjectConstruct) (interface{}, error)

	// Logic
	VisitAnd(expr *And) (interface{}, error)
	VisitNot(expr *Not) (interface{}, error)
	VisitOr(expr *Or) (interface{}, error)

	// Navigation
	VisitElement(expr *Element) (interface{}, error)
	VisitField(expr *Field) (interface{}, error)
	VisitFieldName(expr *FieldName) (interface{}, error)
	VisitSlice(expr *Slice) (interface{}, error)

	// Function
	VisitFunction(expr Function) (interface{}, error)
}
