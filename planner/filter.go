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

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

const (
	FLTR_IS_JOIN     = 1 << iota // is this originally a join filter
	FLTR_IS_ONCLAUSE             // is this an ON-clause filter for ANSI JOIN
	FLTR_IS_DERIVED              // is this a derived filter
	FLTR_IS_UNNEST               // is this ann unnest filter (inherited)
)

type Filter struct {
	fltrExpr  expression.Expression // filter expression
	origExpr  expression.Expression // original filter expression
	keyspaces map[string]bool       // keyspace references
	fltrFlags uint32
}

type Filters []*Filter

func newFilter(fltrExpr, origExpr expression.Expression, keyspaces map[string]bool, isOnclause bool, isJoin bool) *Filter {
	rv := &Filter{
		fltrExpr:  fltrExpr,
		origExpr:  origExpr,
		keyspaces: keyspaces,
	}

	if isOnclause {
		rv.fltrFlags |= FLTR_IS_ONCLAUSE
	}
	if isJoin {
		rv.fltrFlags |= FLTR_IS_JOIN
	}

	return rv
}

func (this *Filter) Copy() *Filter {
	rv := &Filter{
		fltrExpr:  this.fltrExpr.Copy(),
		origExpr:  this.origExpr.Copy(),
		fltrFlags: this.fltrFlags,
	}

	rv.keyspaces = make(map[string]bool, len(this.keyspaces))
	for key, value := range this.keyspaces {
		rv.keyspaces[key] = value
	}

	return rv
}

func (this *Filter) isOnclause() bool {
	return (this.fltrFlags & FLTR_IS_ONCLAUSE) != 0
}

func (this *Filter) isJoin() bool {
	return (this.fltrFlags & FLTR_IS_JOIN) != 0
}

func (this *Filter) isDerived() bool {
	return (this.fltrFlags & FLTR_IS_DERIVED) != 0
}

func (this *Filter) isUnnest() bool {
	return (this.fltrFlags & FLTR_IS_UNNEST) != 0
}

func (this *Filter) setUnnest() {
	this.fltrFlags |= FLTR_IS_UNNEST
}

// check whether the join filter is a single join filter involving any of the keyspaces provided
func (this *Filter) singleJoinFilter(unnestKeyspaces map[string]bool) bool {
	for ks, _ := range this.keyspaces {
		if _, ok := unnestKeyspaces[ks]; !ok {
			return false
		}
	}

	return true
}

// Combine an array of filters into a single expression by ANDing each filter expression,
// perform transformation on each filter, and if an OR filter is involved, perform DNF
// transformation on the combined filter
func combineFilters(baseKeyspace *baseKeyspace, includeOnclause bool) error {
	var err error
	var predHasOr, onHasOr bool
	var dnfPred, origPred, onclause expression.Expression

	for _, fl := range baseKeyspace.filters {
		if fl.isOnclause() {
			if onclause == nil {
				onclause = fl.fltrExpr
			} else {
				onclause = expression.NewAnd(onclause, fl.fltrExpr)
			}

			if _, ok := fl.fltrExpr.(*expression.Or); ok {
				onHasOr = true
			}

			if !includeOnclause {
				continue
			}
		}

		if dnfPred == nil {
			dnfPred = fl.fltrExpr
		} else {
			dnfPred = expression.NewAnd(dnfPred, fl.fltrExpr)
		}

		if fl.origExpr != nil {
			if origPred == nil {
				origPred = fl.origExpr
			} else {
				origPred = expression.NewAnd(origPred, fl.origExpr)
			}
		}

		if _, ok := fl.fltrExpr.(*expression.Or); ok {
			predHasOr = true
		}
	}

	if predHasOr {
		dnf := NewDNF(dnfPred.Copy(), true, true)
		dnfPred, err = dnf.Map(dnfPred)
		if err != nil {
			return err
		}
	}

	if onHasOr {
		dnf := NewDNF(onclause.Copy(), true, true)
		onclause, err = dnf.Map(onclause)
		if err != nil {
			return err
		}
	}

	baseKeyspace.dnfPred = dnfPred
	baseKeyspace.origPred = origPred
	baseKeyspace.onclause = onclause

	return nil
}

// Once a keyspace has been visited, join filters referring to this keyspace can remove
// this keyspace reference since it's now "available", and if there are no other
// keyspace references the join filter can be moved to filters
func (this *builder) moveJoinFilters(keyspace string, baseKeyspace *baseKeyspace) error {
	compact := false
	for i, jfl := range baseKeyspace.joinfilters {
		delete(jfl.keyspaces, keyspace)
		if len(jfl.keyspaces) == 1 {
			for ksName, _ := range jfl.keyspaces {
				if baseKeyspace.name != ksName {
					return errors.NewPlanInternalError(fmt.Sprintf("moveJoinFilters: keyspace mismatch: %s vs %s", baseKeyspace.name, ksName))
				}
				break
			}

			// move to filters
			baseKeyspace.filters = append(baseKeyspace.filters, jfl)
			baseKeyspace.joinfilters[i] = nil
			compact = true
		}
	}

	if compact == true {
		curlen := len(baseKeyspace.joinfilters)
		curlen = trimJoinFilters(baseKeyspace.joinfilters, curlen)
		newlen := curlen
		for i := 0; i < curlen; i++ {
			if i >= newlen {
				break
			}
			if baseKeyspace.joinfilters[i] == nil {
				if i < newlen-1 {
					baseKeyspace.joinfilters[i] = baseKeyspace.joinfilters[newlen-1]
				}
				baseKeyspace.joinfilters[newlen-1] = nil
				newlen = trimJoinFilters(baseKeyspace.joinfilters, newlen-1)
			}
		}
		baseKeyspace.joinfilters = baseKeyspace.joinfilters[:newlen]
	}

	return nil
}

// trim nil entries at the end of joinfilters slice
func trimJoinFilters(joinfilters Filters, curlen int) (newlen int) {
	newlen = curlen
	if newlen == 0 {
		return
	}

	for {
		if joinfilters[newlen-1] == nil {
			newlen--
			if newlen == 0 {
				return
			}
		} else {
			return
		}
	}
}

func (this *builder) processKeyspaceDone(keyspace string) error {
	var err error
	for _, baseKeyspace := range this.baseKeyspaces {
		if baseKeyspace.PlanDone() {
			continue
		} else if keyspace == baseKeyspace.name {
			baseKeyspace.SetPlanDone()
			continue
		}

		err = this.moveJoinFilters(keyspace, baseKeyspace)
		if err != nil {
			return err
		}
	}

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
func deriveNotNullFilter(keyspace datastore.Keyspace, baseKeyspace *baseKeyspace, indexApiVersion int) error {

	// first gather leading index key from all indexes for this keyspace
	indexes := _INDEX_POOL.Get()
	defer _INDEX_POOL.Put(indexes)
	indexes, err := allIndexes(keyspace, nil, indexes, indexApiVersion)
	if err != nil {
		return err
	}

	if len(indexes) == 0 {
		return nil
	}

	formalizer := expression.NewSelfFormalizer(baseKeyspace.name, nil)
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
	for _, fl := range baseKeyspace.filters {
		terms = terms[:0]
		pred := fl.fltrExpr
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
	newFilters := make(Filters, 0, n)
	keyspaceNames := make(map[string]bool, 1)
	keyspaceNames[baseKeyspace.name] = true
	for _, jfl := range baseKeyspace.joinfilters {
		terms = terms[:0]
		pred := jfl.fltrExpr
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
			keyspaces, err := expression.CountKeySpaces(term, keyspaceNames)
			if err != nil {
				return err
			}

			if len(keyspaces) == 0 {
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
					newFilters = addDerivedFilter(term, keyspaceNames, jfl.isOnclause(), newFilters)
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

					min, _, _ := SargableFor(term, expression.Expressions{idxKeyDerive.keyExpr}, false, false)
					if min > 0 {
						keyMap[val].derive = false
						newFilters = addDerivedFilter(term, keyspaceNames, jfl.isOnclause(), newFilters)
					}
				}
			}
		}
	}

	if len(newFilters) > 0 {
		baseKeyspace.filters = append(baseKeyspace.filters, newFilters...)
	}

	return nil
}

func addDerivedFilter(term expression.Expression, keyspaceNames map[string]bool, isOnclause bool, newFilters Filters) Filters {

	newExpr := expression.NewIsNotNull(term)
	newFilter := newFilter(newExpr, newExpr, keyspaceNames, isOnclause, false)
	newFilter.fltrFlags |= FLTR_IS_DERIVED
	newFilters = append(newFilters, newFilter)

	return newFilters
}
