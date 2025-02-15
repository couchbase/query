//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

// Return the filterCovers for a query predicate and index keys. This
// allows array indexes to cover ANY predicates.
func CoversFor(pred, origPred expression.Expression, keys datastore.IndexKeys,
	context *PrepareContext) (map[*expression.Cover]value.Value, error) {

	var fv, ofv map[string]*expression.Cover
	var err error

	fv, err = coversFor(pred, keys, context)
	if err != nil {
		return nil, err
	}
	if origPred != nil {
		ofv, err = coversFor(origPred, keys, context)
		if err != nil {
			return nil, err
		}
	}

	if len(fv)+len(ofv) == 0 {
		return nil, nil
	}

	fc := make(map[*expression.Cover]value.Value, len(fv)+len(ofv))
	for _, v := range fv {
		fc[v] = value.TRUE_VALUE
	}
	for c, ov := range ofv {
		if _, ok := fv[c]; !ok {
			fc[ov] = value.TRUE_VALUE
		}
	}

	return fc, nil

}

func coversFor(pred expression.Expression, keys datastore.IndexKeys,
	context *PrepareContext) (map[string]*expression.Cover, error) {

	cov := &covers{keys, context}
	rv, err := pred.Accept(cov)
	if rv == nil || err != nil {
		return nil, err
	}

	fc := rv.(map[string]*expression.Cover)

	return fc, nil
}

type covers struct {
	keys    datastore.IndexKeys
	context *PrepareContext
}

// Arithmetic

func (this *covers) VisitAdd(expr *expression.Add) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitDiv(expr *expression.Div) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitMod(expr *expression.Mod) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitMult(expr *expression.Mult) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitNeg(expr *expression.Neg) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitSub(expr *expression.Sub) (interface{}, error) {
	return nil, nil
}

// Case

func (this *covers) VisitSearchedCase(expr *expression.SearchedCase) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitSimpleCase(expr *expression.SimpleCase) (interface{}, error) {
	return nil, nil
}

// Collection

func (this *covers) VisitAny(expr *expression.Any) (interface{}, error) {

	for i, k := range this.keys {
		if _, ok := k.Expr.(*expression.All); ok {
			keys := datastore.IndexKeys{k}
			min, _, _, _, _ := SargableFor(expr, nil, nil, keys, nil, (i != 0), true, nil, this.context, nil)
			if min > 0 {
				return map[string]*expression.Cover{
					expr.String(): expression.NewCover(expr),
				}, nil
			}
		}
	}

	return nil, nil
}

func (this *covers) VisitArray(expr *expression.Array) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitEvery(expr *expression.Every) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitAnyEvery(expr *expression.AnyEvery) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitExists(expr *expression.Exists) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitFirst(expr *expression.First) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitObject(expr *expression.Object) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitIn(expr *expression.In) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitWithin(expr *expression.Within) (interface{}, error) {
	return nil, nil
}

// Comparison

func (this *covers) VisitBetween(expr *expression.Between) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitEq(expr *expression.Eq) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitLE(expr *expression.LE) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitLike(expr *expression.Like) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitLT(expr *expression.LT) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitIsMissing(expr *expression.IsMissing) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitIsNotMissing(expr *expression.IsNotMissing) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitIsNotNull(expr *expression.IsNotNull) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitIsNotValued(expr *expression.IsNotValued) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitIsNull(expr *expression.IsNull) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitIsValued(expr *expression.IsValued) (interface{}, error) {
	return nil, nil
}

// Concat
func (this *covers) VisitConcat(expr *expression.Concat) (interface{}, error) {
	return nil, nil
}

// Constant
func (this *covers) VisitConstant(expr *expression.Constant) (interface{}, error) {
	return nil, nil
}

// Identifier
func (this *covers) VisitIdentifier(expr *expression.Identifier) (interface{}, error) {
	return nil, nil
}

// Construction

func (this *covers) VisitArrayConstruct(expr *expression.ArrayConstruct) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitObjectConstruct(expr *expression.ObjectConstruct) (interface{}, error) {
	return nil, nil
}

// Logic

// For AND, return the union over the children
func (this *covers) VisitAnd(expr *expression.And) (interface{}, error) {
	var fc map[string]*expression.Cover

	for _, op := range expr.Operands() {
		oc, err := coversFor(op, this.keys, this.context)
		if err != nil {
			return nil, err
		}

		if len(fc) == 0 {
			fc = oc
		} else {
			for c, v := range oc {
				fc[c] = v
			}
		}
	}

	return fc, nil
}

func (this *covers) VisitNot(expr *expression.Not) (interface{}, error) {
	return nil, nil
}

// For OR, return the intersection over the children
func (this *covers) VisitOr(expr *expression.Or) (interface{}, error) {
	var fc map[string]*expression.Cover

	for _, op := range expr.Operands() {
		oc, err := coversFor(op, this.keys, this.context)
		if err != nil {
			return nil, err
		}

		if len(oc) == 0 {
			return nil, nil
		}

		if fc == nil {
			fc = oc
		} else {
			for c, _ := range fc {
				if _, ok := oc[c]; !ok {
					delete(fc, c)
				}
			}

			if len(fc) == 0 {
				return nil, nil
			}
		}
	}

	return fc, nil
}

// Navigation

func (this *covers) VisitElement(expr *expression.Element) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitField(expr *expression.Field) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitFieldName(expr *expression.FieldName) (interface{}, error) {
	return nil, nil
}

func (this *covers) VisitSlice(expr *expression.Slice) (interface{}, error) {
	return nil, nil
}

// Self
func (this *covers) VisitSelf(expr *expression.Self) (interface{}, error) {
	return nil, nil
}

// Function
func (this *covers) VisitFunction(expr expression.Function) (interface{}, error) {
	return nil, nil
}

// Subquery
func (this *covers) VisitSubquery(expr expression.Subquery) (interface{}, error) {
	return nil, nil
}

// InferUnderParenthesis
func (this *covers) VisitParenInfer(expr expression.ParenInfer) (interface{}, error) {
	return nil, nil
}

// NamedParameter
func (this *covers) VisitNamedParameter(expr expression.NamedParameter) (interface{}, error) {
	return nil, nil
}

// PositionalParameter
func (this *covers) VisitPositionalParameter(expr expression.PositionalParameter) (interface{}, error) {
	return nil, nil
}

// Cover
func (this *covers) VisitCover(expr *expression.Cover) (interface{}, error) {
	return expr.Covered().Accept(this)
}

// All
func (this *covers) VisitAll(expr *expression.All) (interface{}, error) {
	return nil, nil
}
