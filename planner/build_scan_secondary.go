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
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type indexEntry struct {
	index      datastore.Index
	keys       expression.Expressions
	sargKeys   expression.Expressions
	minKeys    int
	sumKeys    int
	cond       expression.Expression
	origCond   expression.Expression
	spans      SargSpans
	exactSpans bool
}

func (this *builder) buildSecondaryScan(indexes map[datastore.Index]*indexEntry,
	node *algebra.KeyspaceTerm, id, pred, limit expression.Expression) (
	plan.SecondaryScan, int, error) {

	if this.cover != nil {
		scan, sargLength, err := this.buildCoveringScan(indexes, node, id, pred, limit)
		if scan != nil || err != nil {
			return scan, sargLength, err
		}
	}

	this.resetCountMin()

	indexes = minimalIndexes(indexes, true)

	var err error
	indexes, err = sargIndexes(indexes, pred)
	if err != nil {
		return nil, 0, err
	}

	// This makes IntersectScan disable limit pushdown, don't use index order
	if this.order != nil && (len(indexes) > 1 || this.maxParallelism > 1) {
		this.resetOrderLimit()
		limit = nil
	}

	var scanBuf [16]plan.SecondaryScan
	var scans []plan.SecondaryScan
	if len(indexes) <= len(scanBuf) {
		scans = scanBuf[0:0]
	} else {
		scans = make([]plan.SecondaryScan, 0, len(indexes))
	}

	sargLength := 0
	var scan plan.SecondaryScan
	for index, entry := range indexes {
		lim := limit

		if this.order != nil {
			if !this.useIndexOrder(entry, entry.keys) {
				this.resetOrderLimit()
				lim = nil
			} else if len(indexes) == 1 {
				this.maxParallelism = 1
			}
		}

		if lim != nil {
			pushDown := false
			arrayIndex := indexHasArrayIndexKey(index)

			if !arrayIndex {
				pushDown, err = allowedPushDown(entry, pred, node.Alias())
				if err != nil {
					return nil, 0, err
				}
			}

			if arrayIndex || !pushDown {
				this.limit = nil
				lim = nil
			}
		}

		scan = entry.spans.CreateScan(index, node, false, pred.MayOverlapSpans(), false, lim, nil, nil)
		scans = append(scans, scan)

		if len(entry.sargKeys) > sargLength {
			sargLength = len(entry.sargKeys)
		}
	}

	if len(scans) == 1 {
		return scans[0], sargLength, nil
	} else {
		return plan.NewIntersectScan(limit, scans...), sargLength, nil
	}
}

func sargableIndexes(indexes []datastore.Index, pred, subset expression.Expression,
	primaryKey expression.Expressions, formalizer *expression.Formalizer) (
	sargables, all, arrays map[datastore.Index]*indexEntry, err error) {

	sargables = make(map[datastore.Index]*indexEntry, len(indexes))
	all = make(map[datastore.Index]*indexEntry, len(indexes))
	arrays = make(map[datastore.Index]*indexEntry, len(indexes))

	var keys expression.Expressions

	for _, index := range indexes {
		isArray := false

		if index.IsPrimary() {
			if primaryKey != nil {
				keys = primaryKey
			} else {
				continue
			}
		} else {
			keys = index.RangeKey()
			keys = keys.Copy()

			for i, key := range keys {
				key = key.Copy()

				key, err = formalizer.Map(key)
				if err != nil {
					return
				}

				dnf := NewDNF(key)
				key, err = dnf.Map(key)
				if err != nil {
					return
				}

				keys[i] = key

				if !isArray {
					isArray, _ = key.IsArrayIndexKey()
				}
			}
		}

		var origCond expression.Expression
		cond := index.Condition()
		if cond != nil {
			if subset == nil {
				continue
			}

			cond = cond.Copy()

			cond, err = formalizer.Map(cond)
			if err != nil {
				return
			}

			origCond = cond.Copy()

			dnf := NewDNF(cond)
			cond, err = dnf.Map(cond)
			if err != nil {
				return
			}

			if !SubsetOf(subset, cond) {
				continue
			}
		}

		min, sum := SargableFor(pred, keys)
		entry := &indexEntry{
			index, keys, keys[0:min], min, sum, cond, origCond, nil, false,
		}
		all[index] = entry

		if min > 0 {
			sargables[index] = entry
		}

		if isArray {
			arrays[index] = entry
		}
	}

	return sargables, all, arrays, nil
}

func minimalIndexes(sargables map[datastore.Index]*indexEntry, shortest bool) map[datastore.Index]*indexEntry {

	for s, se := range sargables {
		for t, te := range sargables {
			if t == s {
				continue
			}

			if narrowerOrEquivalent(se, te, shortest) {
				delete(sargables, t)
			}
		}
	}

	return sargables
}

/*
Is se narrower or equivalent to te.
*/
func narrowerOrEquivalent(se, te *indexEntry, shortest bool) bool {
	if len(te.sargKeys) > len(se.sargKeys) {
		return false
	}

	if te.cond != nil && (se.cond == nil || !SubsetOf(se.cond, te.cond)) {
		return false
	}

	var fc map[string]value.Value
	if se.cond != nil {
		fc = _FILTER_COVERS_POOL.Get()
		defer _FILTER_COVERS_POOL.Put(fc)
		fc = se.cond.FilterCovers(fc)
	}
outer:
	for _, tk := range te.sargKeys {
		for _, sk := range se.sargKeys {
			if SubsetOf(sk, tk) || sk.DependsOn(tk) {
				continue outer
			}
		}

		if se.cond == nil {
			return false
		}

		if _, ok := fc[tk.String()]; !ok {
			return false
		}
	}

	return se.sumKeys > te.sumKeys ||
		(shortest && (len(se.keys) <= len(te.keys)))
}

func sargIndexes(sargables map[datastore.Index]*indexEntry, pred expression.Expression) (
	map[datastore.Index]*indexEntry, error) {
	for _, se := range sargables {
		spans, exactSpans, err := SargFor(pred, se.keys, se.minKeys, len(se.keys))
		if err != nil || spans.Size() == 0 {
			logging.Errorp("Sargable index not sarged", logging.Pair{"pred", pred},
				logging.Pair{"sarg_keys", se.sargKeys}, logging.Pair{"error", err})
			return nil, errors.NewPlanError(nil, fmt.Sprintf("Sargable index not sarged; pred=%v, sarg_keys=%v, error=%v",
				pred.String(), se.sargKeys.String(), err))
			return nil, err
		}

		se.spans = spans
		se.exactSpans = exactSpans
	}

	return sargables, nil
}

func (this *builder) useIndexOrder(entry *indexEntry, keys expression.Expressions) bool {
	if !entry.spans.CanUseIndexOrder() {
		return false
	}

	var filters map[string]value.Value
	if entry.cond != nil {
		filters = _FILTER_COVERS_POOL.Get()
		defer _FILTER_COVERS_POOL.Put(filters)
		filters = entry.cond.FilterCovers(filters)
	}

	i := 0
outer:
	for _, orderTerm := range this.order.Terms() {
		// orderTerm is constant
		if orderTerm.Expression().Static() != nil {
			continue
		}

		// non-constant orderTerms are more than index keys
		if i >= len(keys) {
			// match with condition EQ terms
			if equalConditionFilter(filters, orderTerm.Expression().String()) {
				continue outer
			}
			return false
		}

		if orderTerm.Descending() {
			return false
		}

		if isArray, _ := entry.keys[i].IsArrayIndexKey(); isArray {
			return false
		}

		for {
			if orderTerm.Expression().EquivalentTo(keys[i]) {
				// orderTerm matched with index key
				i++
				continue outer
			} else if equalConditionFilter(filters, orderTerm.Expression().String()) {
				// orderTerm matched with Condition EQ
				continue outer
			} else if eq, _ := entry.spans.EquivalenceRangeAt(i); eq {
				// orderTerm not yet matched, but can skip equivalence range key
				i++
				if i >= len(keys) {
					return false
				}
			} else {
				return false
			}
		}
	}

	return true
}

func equalConditionFilter(filters map[string]value.Value, str string) bool {
	if filters == nil {
		return false
	}

	v, ok := filters[str]
	return ok && v != nil
}

func allowedPushDown(entry *indexEntry, pred expression.Expression, alias string) (
	bool, error) {

	if !entry.exactSpans {
		return false, nil
	}

	// check for non sargable key is in predicate
	exprs, _, err := indexCoverExpressions(entry, entry.sargKeys, pred)
	if err != nil {
		return false, err
	}

	return pred.CoveredBy(alias, exprs), nil
}

func indexHasArrayIndexKey(index datastore.Index) bool {
	for _, sk := range index.RangeKey() {
		if isArray, _ := sk.IsArrayIndexKey(); isArray {
			return true
		}
	}
	return false
}
