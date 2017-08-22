//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"fmt"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

// breaks expr on AND boundaries and classify into appropriate keyspaces
func ClassifyExpr(expr expression.Expression, baseKeyspaces map[string]*baseKeyspace) error {

	if len(baseKeyspaces) == 0 {
		return errors.NewPlanError(nil, "ClassifyExpr: invalid argument baseKeyspaces")
	}

	classifier := newExprClassifier(baseKeyspaces)
	_, err := expr.Accept(classifier)
	if err != nil {
		return err
	}

	return nil
}

type exprClassifier struct {
	baseKeyspaces map[string]*baseKeyspace
	keyspaceNames map[string]bool
}

func newExprClassifier(baseKeyspaces map[string]*baseKeyspace) *exprClassifier {

	rv := &exprClassifier{
		baseKeyspaces: baseKeyspaces,
	}

	rv.keyspaceNames = make(map[string]bool, len(baseKeyspaces))
	for _, keyspace := range baseKeyspaces {
		rv.keyspaceNames[keyspace.name] = true
	}

	return rv
}

func (this *exprClassifier) VisitAnd(expr *expression.And) (interface{}, error) {

	var err error
	for _, op := range expr.Operands() {
		switch op := op.(type) {
		case *expression.And:
			_, err = this.VisitAnd(op)
		default:
			_, err = this.visitDefault(op)
		}
		if err != nil {
			return nil, err
		}
	}

	return expr, nil
}

// Arithmetic

func (this *exprClassifier) VisitAdd(pred *expression.Add) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitDiv(pred *expression.Div) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitMod(pred *expression.Mod) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitMult(pred *expression.Mult) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitNeg(pred *expression.Neg) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitSub(pred *expression.Sub) (interface{}, error) {
	return this.visitDefault(pred)
}

// Case

func (this *exprClassifier) VisitSearchedCase(pred *expression.SearchedCase) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitSimpleCase(pred *expression.SimpleCase) (interface{}, error) {
	return this.visitDefault(pred)
}

// Collection

func (this *exprClassifier) VisitAny(pred *expression.Any) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitEvery(pred *expression.Every) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitAnyEvery(pred *expression.AnyEvery) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitArray(pred *expression.Array) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitFirst(pred *expression.First) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitObject(pred *expression.Object) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitExists(pred *expression.Exists) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitIn(pred *expression.In) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitWithin(pred *expression.Within) (interface{}, error) {
	return this.visitDefault(pred)
}

// Comparison

func (this *exprClassifier) VisitBetween(pred *expression.Between) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitEq(pred *expression.Eq) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitLE(pred *expression.LE) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitLike(pred *expression.Like) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitLT(pred *expression.LT) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitIsMissing(pred *expression.IsMissing) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitIsNotMissing(pred *expression.IsNotMissing) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitIsNotNull(pred *expression.IsNotNull) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitIsNotValued(pred *expression.IsNotValued) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitIsNull(pred *expression.IsNull) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitIsValued(pred *expression.IsValued) (interface{}, error) {
	return this.visitDefault(pred)
}

// Concat
func (this *exprClassifier) VisitConcat(pred *expression.Concat) (interface{}, error) {
	return this.visitDefault(pred)
}

// Constant
func (this *exprClassifier) VisitConstant(pred *expression.Constant) (interface{}, error) {
	return this.visitDefault(pred)
}

// Identifier
func (this *exprClassifier) VisitIdentifier(pred *expression.Identifier) (interface{}, error) {
	return this.visitDefault(pred)
}

// Construction

func (this *exprClassifier) VisitArrayConstruct(pred *expression.ArrayConstruct) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitObjectConstruct(pred *expression.ObjectConstruct) (interface{}, error) {
	return this.visitDefault(pred)
}

// Logic

func (this *exprClassifier) VisitNot(pred *expression.Not) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitOr(pred *expression.Or) (interface{}, error) {
	return this.visitDefault(pred)
}

// Navigation

func (this *exprClassifier) VisitElement(pred *expression.Element) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitField(pred *expression.Field) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitFieldName(pred *expression.FieldName) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitSlice(pred *expression.Slice) (interface{}, error) {
	return this.visitDefault(pred)
}

// Self
func (this *exprClassifier) VisitSelf(pred *expression.Self) (interface{}, error) {
	return this.visitDefault(pred)
}

// Function
func (this *exprClassifier) VisitFunction(pred expression.Function) (interface{}, error) {
	return this.visitDefault(pred)
}

// Subquery
func (this *exprClassifier) VisitSubquery(pred expression.Subquery) (interface{}, error) {
	return this.visitDefault(pred)
}

// NamedParameter
func (this *exprClassifier) VisitNamedParameter(pred expression.NamedParameter) (interface{}, error) {
	return this.visitDefault(pred)
}

// PositionalParameter
func (this *exprClassifier) VisitPositionalParameter(pred expression.PositionalParameter) (interface{}, error) {
	return this.visitDefault(pred)
}

// Cover
func (this *exprClassifier) VisitCover(pred *expression.Cover) (interface{}, error) {
	return this.visitDefault(pred)
}

// All
func (this *exprClassifier) VisitAll(pred *expression.All) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) visitDefault(expr expression.Expression) (interface{}, error) {

	cpred := expr.Value()
	if cpred != nil && cpred.Truth() {
		return expr, nil
	}

	keyspaces, err := expression.CountKeySpaces(expr, this.keyspaceNames)
	if err != nil {
		return nil, err
	}

	if len(keyspaces) < 1 {
		return expr, nil
	}

	filter := newFilter(expr, keyspaces)

	for kspace, _ := range keyspaces {
		if baseKspace, ok := this.baseKeyspaces[kspace]; ok {
			if len(keyspaces) == 1 {
				baseKspace.filters = append(baseKspace.filters, filter)
			} else {
				baseKspace.joinfilters = append(baseKspace.joinfilters, filter)
				// if this is an OR join predicate, attempt to extract a new OR-predicate
				// for a single keyspace (to enable union scan)
				if or, ok := expr.(*expression.Or); ok {
					newpred, err := this.extractExpr(or, baseKspace.name)
					if err != nil {
						return nil, err
					}
					if newpred != nil {
						newKeyspaces := make(map[string]bool)
						newKeyspaces[baseKspace.name] = true
						newfilter := newFilter(newpred, newKeyspaces)
						baseKspace.filters = append(baseKspace.filters, newfilter)
					}
				}
			}
		} else {
			return nil, errors.NewPlanInternalError(fmt.Sprintf("exprClassifier.visitDefault: missing keyspace %s", kspace))
		}
	}

	return expr, nil
}

func (this *exprClassifier) extractExpr(or *expression.Or, basekspace string) (expression.Expression, error) {
	orTerms, truth := flattenOr(or)
	if orTerms == nil || truth {
		return nil, nil
	}

	var newTerm expression.Expression
	var newTerms expression.Expressions = nil
	for _, op := range orTerms.Operands() {
		baseKeyspaces := copyBaseKeyspaces(this.baseKeyspaces)
		err := ClassifyExpr(op, baseKeyspaces)
		if err != nil {
			return nil, err
		}
		newTerm = nil
		if kspace, ok := baseKeyspaces[basekspace]; ok {
			for _, fl := range kspace.filters {
				if newTerm == nil {
					newTerm = fl.fltrexpr
				} else {
					newTerm = expression.NewAnd(newTerm, fl.fltrexpr)
				}
			}
		}

		if newTerm != nil {
			newTerms = append(newTerms, newTerm)
		} else {
			return nil, nil
		}
	}

	return expression.NewOr(newTerms...), nil
}
