//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	base "github.com/couchbase/query/plannerbase"
)

// Combine an array of filters into a single expression by ANDing each filter expression,
// perform transformation on each filter, and if an OR filter is involved, perform DNF
// transformation on the combined filter
func CombineFilters(baseKeyspace *base.BaseKeyspace, includeOnclause, onclauseOnly bool) error {
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
		} else {
			if onclauseOnly {
				continue
			}
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
func deriveNotNullFilter(keyspace datastore.Keyspace, baseKeyspace *base.BaseKeyspace, useCBO bool,
	indexApiVersion int, virtualIndexes []datastore.Index, advisorValidate bool,
	context *PrepareContext) error {

	// first gather leading index key from all indexes for this keyspace
	indexes := _INDEX_POOL.Get()
	defer _INDEX_POOL.Put(indexes)
	indexes, err := allIndexes(keyspace, nil, indexes, indexApiVersion, false)
	if err != nil {
		return err
	}

	if len(virtualIndexes) > 0 {
		indexes = append(indexes, virtualIndexes...)
	}

	if len(indexes) == 0 {
		return nil
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
			isArray, _ := key.IsArrayIndexKey()
			if isArray {
				continue
			}

			key = key.Copy()
			key, err = formalizer.Map(key)
			if err != nil {
				return err
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
		return nil
	}

	// next check existing filters
	terms := make(expression.Expressions, 0, 3)
	for _, fl := range baseKeyspace.Filters() {
		terms = terms[:0]
		pred := fl.FltrExpr()
		if not, ok := pred.(*expression.Not); ok {
			pred = not.Operand()
		}
		switch pred := pred.(type) {
		case *expression.IsNotMissing:
			terms = append(terms, pred.Operand())
		case *expression.IsNotNull:
			terms = append(terms, pred.Operand())
		case *expression.IsValued:
			terms = append(terms, pred.Operand())
		case *expression.Eq:
			terms = append(terms, pred.First(), pred.Second())
		case *expression.LE:
			terms = append(terms, pred.First(), pred.Second())
		case *expression.LT:
			terms = append(terms, pred.First(), pred.Second())
		case *expression.Like:
			terms = append(terms, pred.First(), pred.Second())
		case *expression.Between:
			terms = append(terms, pred.First(), pred.Second(), pred.Third())
		}

		for _, term := range terms {
			val := term.String()
			if val == "" {
				continue
			}
			if _, ok := keyMap[val]; ok {
				keyMap[val].derive = false
				n--
				if n == 0 {
					return nil
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
		terms = terms[:0]
		pred := jfl.FltrExpr()
		if not, ok := pred.(*expression.Not); ok {
			pred = not.Operand()
		}
		switch pred := pred.(type) {
		case *expression.Eq:
			terms = append(terms, pred.First(), pred.Second())
		case *expression.LE:
			terms = append(terms, pred.First(), pred.Second())
		case *expression.LT:
			terms = append(terms, pred.First(), pred.Second())
		case *expression.Like:
			terms = append(terms, pred.First(), pred.Second())
		case *expression.Between:
			terms = append(terms, pred.First(), pred.Second(), pred.Third())
		}

		for _, term := range terms {
			// check whether the expression references current keyspace
			if !expression.HasKeyspaceReferences(term, keyspaceNames) {
				continue
			}

			val := term.String()
			if val == "" {
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
						context)
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

					min, _, _, _ := SargableFor(term, expression.Expressions{idxKeyDerive.keyExpr}, false, false)
					if min > 0 {
						keyMap[val].derive = false
						newFilters = AddDerivedFilter(term, keyspaceNames, origKeyspaceNames,
							jfl.IsOnclause(), newFilters, useCBO, advisorValidate,
							context)
					}
				}
			}
		}
	}

	if len(newFilters) > 0 {
		baseKeyspace.AddFilters(newFilters)
	}

	return nil
}

func AddDerivedFilter(term expression.Expression, keyspaceNames, origKeyspaceNames map[string]string,
	isOnclause bool, newFilters base.Filters, useCBO, advisorValidate bool,
	context *PrepareContext) base.Filters {

	newExpr := expression.NewIsNotNull(term)
	newFilter := base.NewFilter(newExpr, newExpr, keyspaceNames, origKeyspaceNames, isOnclause, false)
	newFilter.SetDerived()
	if useCBO {
		selec, _ := optExprSelec(origKeyspaceNames, newExpr, advisorValidate, context)
		newFilter.SetSelec(selec)
	}
	newFilters = append(newFilters, newFilter)

	return newFilters
}
