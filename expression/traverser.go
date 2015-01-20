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

func (this *TraverserBase) VisitArray(expr *Array) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitFirst(expr *First) (interface{}, error) {
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

// Function

func (this *TraverserBase) VisitFunction(expr Function) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

// Subquery
func (this *TraverserBase) VisitSubquery(expr Subquery) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

// Parameters

func (this *TraverserBase) VisitNamedParameter(expr NamedParameter) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

func (this *TraverserBase) VisitPositionalParameter(expr PositionalParameter) (interface{}, error) {
	return nil, this.TraverseList(expr.Children())
}

// Init
func (this *TraverserBase) SetTraverser(traverser Traverser) {
	if this.traverser == nil {
		this.traverser = traverser
	}
}
