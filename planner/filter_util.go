//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	base "github.com/couchbase/query/plannerbase"
)

// Combine an array of filters into a single expression by ANDing each filter expression,
// perform transformation on each filter, and if an OR filter is involved, perform DNF
// transformation on the combined filter
func CombineFilters(baseKeyspace *base.BaseKeyspace, includeOnclause bool) error {
	var err error
	var predHasOr, onHasOr bool
	var dnfPred, origPred, onclause expression.Expression

	for _, fl := range baseKeyspace.Filters() {
		fltrExpr := fl.FltrExpr()
		origExpr := fl.OrigExpr()

		if fl.IsOnclause() {
			if onclause == nil {
				onclause = fltrExpr
			} else {
				onclause = expression.NewAnd(onclause, fltrExpr)
			}

			if _, ok := fltrExpr.(*expression.Or); ok {
				onHasOr = true
			}

			if !includeOnclause {
				continue
			}

			if baseKeyspace.IsOuter() && fl.NotPushable() {
				continue
			}
		} else {
			// MB-38564, MB-46607: in case of outer join, filters from the
			// WHERE clause should not be pushed to a subservient table
			if baseKeyspace.OnclauseOnly() || baseKeyspace.IsOuter() {
				continue
			}
		}

		// do not include vector function filter by default
		if fl.IsVectorFunc() {
			continue
		}

		if dnfPred == nil {
			dnfPred = fltrExpr
		} else {
			dnfPred = expression.NewAnd(dnfPred, fltrExpr)
		}

		if origExpr != nil {
			if origPred == nil {
				origPred = origExpr
			} else {
				origPred = expression.NewAnd(origPred, origExpr)
			}
		}

		if _, ok := fltrExpr.(*expression.Or); ok {
			predHasOr = true
		}
	}

	if predHasOr {
		dnf := base.NewDNF(dnfPred.Copy(), true, true)
		dnfPred, err = dnf.Map(dnfPred)
		if err != nil {
			return err
		}
	}

	if onHasOr {
		dnf := base.NewDNF(onclause.Copy(), true, true)
		onclause, err = dnf.Map(onclause)
		if err != nil {
			return err
		}
	}

	baseKeyspace.SetPreds(dnfPred, origPred, onclause)

	return nil
}

type idxKeyDerive struct {
	keyExpr expression.Expression // index key expression
	derive  bool                  // need to derive?
}

func newIdxKeyDerive(keyExpr expression.Expression) *idxKeyDerive {
	return &idxKeyDerive{
		keyExpr: keyExpr,
		derive:  true,
	}
}

// derive IS NOT NULL filters for a keyspace based on join filters in the
// WHERE clause as well as ON-clause of inner joins
func (this *builder) deriveNotNullFilter(keyspace datastore.Keyspace, baseKeyspace *base.BaseKeyspace,
	useCBO bool, virtualIndexes []datastore.Index, advisorValidate bool,
	context *PrepareContext, aliases map[string]bool) (error, time.Duration) {

	// first gather leading index key from all indexes for this keyspace
	indexes, err, duration := allIndexes(keyspace, nil, virtualIndexes, false, context)
	if nil != indexes {
		defer _INDEX_POOL.Put(indexes)
	}
	if err != nil {
		return err, duration
	}

	if len(indexes) == 0 {
		return nil, duration
	}

	if useCBO && baseKeyspace.DocCount() < 0 {
		useCBO = false
	}

	formalizer := expression.NewSelfFormalizer(baseKeyspace.Name(), nil)
	keyMap := make(map[string]*idxKeyDerive, len(indexes))

	for _, index := range indexes {
		if index.IsPrimary() {
			continue
		}

		keys := index.RangeKey()
		if len(keys) > 0 {
			key := keys[0]
			isArray, _, _ := key.IsArrayIndexKey()
			if isArray {
				continue
			}

			key = key.Copy()
			formalizer.SetIndexScope()
			key, err = formalizer.Map(key)
			formalizer.ClearIndexScope()
			if err != nil {
				return err, duration
			}

			val := key.String()
			if val == "" {
				continue
			}

			if _, ok := keyMap[val]; !ok {
				keyMap[val] = newIdxKeyDerive(key)
			}
		}
	}

	n := len(keyMap)

	// in case only primary index or index with leading array index key exists
	if n == 0 {
		return nil, duration
	}

	// next check existing filters
	terms := make(map[string]expression.Expression, 8)
	for _, fl := range baseKeyspace.Filters() {
		// clear terms
		for k, _ := range terms {
			delete(terms, k)
		}

		pred := fl.FltrExpr()
		if not, ok := pred.(*expression.Not); ok {
			pred = not.Operand()
		}

		getTerms(pred, terms, false)

		for val, _ := range terms {
			if _, ok := keyMap[val]; ok {
				keyMap[val].derive = false
				n--
				if n == 0 {
					return nil, duration
				}
			}
		}
	}

	// next check all join filters
	newFilters := make(base.Filters, 0, n)
	keyspaceNames := make(map[string]string, 1)
	keyspaceNames[baseKeyspace.Name()] = baseKeyspace.Keyspace()
	origKeyspaceNames := make(map[string]string, 1)
	origKeyspaceNames[baseKeyspace.Name()] = baseKeyspace.Keyspace()
	for _, jfl := range baseKeyspace.JoinFilters() {
		// clear terms
		for k, _ := range terms {
			delete(terms, k)
		}

		pred := jfl.FltrExpr()
		if not, ok := pred.(*expression.Not); ok {
			pred = not.Operand()
		}

		getTerms(pred, terms, true)

		for val, term := range terms {
			// check whether the expression references current keyspace
			if !expression.HasKeyspaceReferences(term, keyspaceNames) {
				continue
			}

			// if the expression is an index leading key, and there is no
			// filter yet that reference this expression, derive a new
			// IS NOT NULL expression
			if _, ok := keyMap[val]; ok {
				if keyMap[val].derive == false {
					continue
				} else {
					keyMap[val].derive = false
					newFilters = AddDerivedFilter(term, keyspaceNames, origKeyspaceNames,
						jfl.IsOnclause(), newFilters, useCBO, advisorValidate,
						baseKeyspace.OptBit(), context)
				}
			} else {
				simple := false
				if _, ok := term.(*expression.Field); ok {
					simple = true
				} else if _, ok := term.(*expression.Identifier); ok {
					simple = true
				}

				// if the term expression is a simple expression, no need to check
				// further for sargable
				if simple {
					continue
				}

				// check all indexes for sargable
				for val, idxKeyDerive := range keyMap {
					if idxKeyDerive.derive == false {
						continue
					}

					// the "gsi" argument is for skip index keys; we don't need it
					// here since we only consider the leading index key
					keys := datastore.IndexKeys{&datastore.IndexKey{idxKeyDerive.keyExpr, datastore.IK_NONE}}
					min, _, _, _, _ := SargableFor(term, nil, nil, keys, nil, false, false, nil, context, aliases)
					if min > 0 {
						keyMap[val].derive = false
						idxKey := term
						if !expression.HasSingleKeyspaceReference(term, baseKeyspace.Name(), this.keyspaceNames) {
							// in case the term expression references more
							// than just this keyspace, use the leading
							// index key for derivation purpose
							idxKey = idxKeyDerive.keyExpr
						}
						newFilters = AddDerivedFilter(idxKey, keyspaceNames, origKeyspaceNames,
							jfl.IsOnclause(), newFilters, useCBO, advisorValidate,
							baseKeyspace.OptBit(), context)
					}
				}
			}
		}
	}

	if len(newFilters) > 0 {
		baseKeyspace.AddFilters(newFilters)
	}

	return nil, duration
}

func AddDerivedFilter(term expression.Expression, keyspaceNames, origKeyspaceNames map[string]string,
	isOnclause bool, newFilters base.Filters, useCBO, advisorValidate bool,
	optBit int32, context *PrepareContext) base.Filters {

	newExpr := expression.NewIsNotNull(term)
	newExpr.SetExprFlag(expression.EXPR_JOIN_NOT_NULL)
	newFilter := base.NewFilter(newExpr, newExpr, keyspaceNames, origKeyspaceNames, isOnclause, false)
	newFilter.SetDerived()
	if useCBO {
		optFilterSelectivity(newFilter, advisorValidate, context)
		newFilter.SetOptBits(optBit)
	}
	newFilters = append(newFilters, newFilter)

	return newFilters
}

func getOptBits(baseKeyspaces map[string]*base.BaseKeyspace, keyspaces map[string]string) int32 {
	bits := int32(0)
	for a, _ := range keyspaces {
		if ks, ok := baseKeyspaces[a]; ok {
			bits |= ks.OptBit()
		}
	}
	return bits
}

func addTerm(terms map[string]expression.Expression, preds ...expression.Expression) {
	for _, pred := range preds {
		str := pred.String()
		if str != "" {
			if _, ok := terms[str]; !ok {
				terms[str] = pred
			}
		}
	}
}

func getTerms(pred expression.Expression, terms map[string]expression.Expression, join bool) {
	switch pred := pred.(type) {
	case *expression.IsNotMissing:
		if !join {
			addTerm(terms, pred.Operand())
		}
	case *expression.IsNotNull:
		if !join {
			addTerm(terms, pred.Operand())
		}
	case *expression.IsValued:
		if !join {
			addTerm(terms, pred.Operand())
		}
	case *expression.Eq:
		addTerm(terms, pred.First(), pred.Second())
	case *expression.LE:
		addTerm(terms, pred.First(), pred.Second())
	case *expression.LT:
		addTerm(terms, pred.First(), pred.Second())
	case *expression.Like:
		addTerm(terms, pred.First(), pred.Second())
	case *expression.Between:
		addTerm(terms, pred.First(), pred.Second(), pred.Third())
	case *expression.In:
		addTerm(terms, pred.First())
	case *expression.And:
		// for And under Or
		for _, child := range pred.Operands() {
			getTerms(child, terms, join)
		}
	case *expression.Or:
		orTerms := getOrclauseTerms(pred)
		for str, term := range orTerms {
			if _, ok := terms[str]; !ok {
				terms[str] = term
			}
		}
	}
}

func getOrclauseTerms(or *expression.Or) map[string]expression.Expression {
	newOr, _ := expression.FlattenOrNoDedup(or)
	allTerms := make(map[string]expression.Expression)
	tempTerms := make(map[string]expression.Expression)
	for i, child := range newOr.Operands() {
		var terms map[string]expression.Expression
		if i == 0 {
			terms = allTerms
		} else {
			for k, _ := range tempTerms {
				delete(tempTerms, k)
			}
			terms = tempTerms
		}

		// use false even for join filters since an arm of or may not be join filterx
		getTerms(child, terms, false)

		if i > 0 {
			// only keep common terms
			for k, _ := range allTerms {
				if _, ok := tempTerms[k]; !ok {
					delete(allTerms, k)
				}
			}
			if len(allTerms) == 0 {
				return nil
			}
		}
	}
	return allTerms
}
