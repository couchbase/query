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
	base "github.com/couchbase/query/plannerbase"
)

type sarg struct {
	key             expression.Expression
	baseKeyspace    *base.BaseKeyspace
	keyspaceNames   map[string]string
	isJoin          bool
	doSelec         bool
	advisorValidate bool
	constPred       bool
	isMissing       bool
	isArray         bool
	isVector        bool
	index           datastore.Index
	keyPos          int
	aliases         map[string]bool
	context         *PrepareContext
}

func newSarg(key expression.Expression, index datastore.Index, baseKeyspace *base.BaseKeyspace,
	keyspaceNames map[string]string, isJoin, doSelec, advisorValidate, isMissing, isArray, isVector bool,
	keyPos int, aliases map[string]bool, context *PrepareContext) *sarg {
	return &sarg{
		key:             key,
		baseKeyspace:    baseKeyspace,
		keyspaceNames:   keyspaceNames,
		isJoin:          isJoin,
		doSelec:         doSelec,
		advisorValidate: advisorValidate,
		isMissing:       isMissing,
		isArray:         isArray,
		isVector:        isVector,
		index:           index,
		keyPos:          keyPos,
		aliases:         aliases,
		context:         context,
	}
}

func (this *sarg) getSarg(pred expression.Expression) expression.Expression {
	if pred == nil {
		return nil
	}

	cpred := pred.Static()
	if cpred != nil {
		return cpred
	} else if !this.isJoin {
		if !this.baseKeyspace.IsInCorrSubq() {
			return nil
		} else if expression.HasKeyspaceReferences(pred, this.keyspaceNames) {
			// in correlated subquery, check whether it references any other
			// keyspaces in current query block (joins)
			return nil
		}
	}

	if pred.Indexable() {
		// make sure the expression does NOT reference current keyspace
		keyspaceNames := make(map[string]string, 1+len(this.baseKeyspace.GetUnnests()))
		keyspaceNames[this.baseKeyspace.Name()] = this.baseKeyspace.Keyspace()
		for a, k := range this.baseKeyspace.GetUnnests() {
			keyspaceNames[a] = k
		}
		if !expression.HasKeyspaceReferences(pred, keyspaceNames) {
			return pred.Copy()
		}
	}

	return nil
}

func (this *sarg) getSelec(pred expression.Expression) float64 {
	if !this.doSelec {
		return OPT_SELEC_NOT_AVAIL
	}

	var array bool
	switch pred.(type) {
	case *expression.Any, *expression.AnyEvery, *expression.Every:
		array = true
	}

	for _, fl := range this.baseKeyspace.Filters() {
		if pred.EquivalentTo(fl.FltrExpr()) {
			if array {
				return fl.ArraySelec()
			}
			return fl.Selec()
		}
	}

	// if this is a subterm of an OR, it won't be in filters
	var keyspaces map[string]string
	var err error
	if this.isJoin {
		keyspaces, err = expression.CountKeySpaces(pred, this.keyspaceNames)
		if err != nil {
			return OPT_SELEC_NOT_AVAIL
		}
	} else {
		keyspaces = make(map[string]string, 1)
		keyspaces[this.baseKeyspace.Name()] = this.baseKeyspace.Keyspace()
	}
	sel, arrSel := optExprSelec(keyspaces, pred, this.advisorValidate, this.context)

	if array {
		return arrSel
	}
	return sel
}

// Arithmetic

func (this *sarg) VisitAdd(pred *expression.Add) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitDiv(pred *expression.Div) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitMod(pred *expression.Mod) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitMult(pred *expression.Mult) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitNeg(pred *expression.Neg) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitSub(pred *expression.Sub) (interface{}, error) {
	return this.visitDefault(pred)
}

// Case

func (this *sarg) VisitSearchedCase(pred *expression.SearchedCase) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitSimpleCase(pred *expression.SimpleCase) (interface{}, error) {
	return this.visitDefault(pred)
}

// Collection

func (this *sarg) VisitArray(pred *expression.Array) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitEvery(pred *expression.Every) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitExists(pred *expression.Exists) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitFirst(pred *expression.First) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitObject(pred *expression.Object) (interface{}, error) {
	return this.visitDefault(pred)
}

// Comparison

func (this *sarg) VisitBetween(pred *expression.Between) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitLike(pred *expression.Like) (interface{}, error) {
	return this.visitLike(pred)
}

// Concat
func (this *sarg) VisitConcat(pred *expression.Concat) (interface{}, error) {
	return this.visitDefault(pred)
}

// Constant
func (this *sarg) VisitConstant(pred *expression.Constant) (interface{}, error) {
	val := pred.Value()
	if val == nil || !val.Truth() {
		// mark if it is not a TRUE constant (TRUE constant does not introduce false positives)
		this.constPred = true
	}
	return this.visitDefault(pred)
}

// Identifier
func (this *sarg) VisitIdentifier(pred *expression.Identifier) (interface{}, error) {
	return this.visitDefault(pred)
}

// Construction

func (this *sarg) VisitArrayConstruct(pred *expression.ArrayConstruct) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitObjectConstruct(pred *expression.ObjectConstruct) (interface{}, error) {
	return this.visitDefault(pred)
}

// Logic

func (this *sarg) VisitNot(pred *expression.Not) (interface{}, error) {
	return this.visitDefault(pred)
}

// Navigation

func (this *sarg) VisitElement(pred *expression.Element) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitField(pred *expression.Field) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitFieldName(pred *expression.FieldName) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *sarg) VisitSlice(pred *expression.Slice) (interface{}, error) {
	return this.visitDefault(pred)
}

// Self
func (this *sarg) VisitSelf(pred *expression.Self) (interface{}, error) {
	return this.visitDefault(pred)
}

// Function
func (this *sarg) VisitFunction(pred expression.Function) (interface{}, error) {
	switch pred := pred.(type) {
	case *expression.RegexpLike:
		return this.visitLike(pred)
	case *expression.Ann:
		if this.isVector {
			if index6, ok := this.index.(datastore.Index6); ok {
				fld := pred.Field()
				if fld.EquivalentTo(this.key) &&
					datastore.CompatibleMetric(index6.VectorDistanceType(), pred.Metric()) {
					rv := _WHOLE_SPANS.Copy().(*TermSpans)
					rv.ann = pred
					rv.annPos = this.keyPos
					return rv, nil
				}
			}
		}
		return nil, nil
	}

	return this.visitDefault(pred)
}

// Subquery
func (this *sarg) VisitSubquery(pred expression.Subquery) (interface{}, error) {
	return this.visitDefault(pred)
}

// InferUnderParenthesis
func (this *sarg) VisitParenInfer(pred expression.ParenInfer) (interface{}, error) {
	return this.visitDefault(pred)
}

// NamedParameter
func (this *sarg) VisitNamedParameter(pred expression.NamedParameter) (interface{}, error) {
	this.constPred = true
	return this.visitDefault(pred)
}

// PositionalParameter
func (this *sarg) VisitPositionalParameter(pred expression.PositionalParameter) (interface{}, error) {
	this.constPred = true
	return this.visitDefault(pred)
}

// Cover
func (this *sarg) VisitCover(pred *expression.Cover) (interface{}, error) {
	return pred.Covered().Accept(this)
}

// All
func (this *sarg) VisitAll(pred *expression.All) (interface{}, error) {
	return pred.Array().Accept(this)
}
