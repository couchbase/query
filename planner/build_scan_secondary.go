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
	cond       expression.Expression
	origCond   expression.Expression
	spans      plan.Spans
	exactSpans bool
}

func (this *builder) buildSecondaryScan(indexes map[datastore.Index]*indexEntry,
	node *algebra.KeyspaceTerm, id, pred, limit expression.Expression) (
	plan.Operator, int, error) {

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

	if (this.order != nil || limit != nil) && len(indexes) > 1 {
		// This makes IntersectScan disable limit pushdown, don't use index order
		this.resetOrderLimit()
		limit = nil
	}
	if this.order != nil && this.maxParallelism > 1 {
		this.resetOrderLimit()
		limit = nil
	}

	var scanBuf [16]plan.Operator
	var scans []plan.Operator
	if len(indexes) <= len(scanBuf) {
		scans = scanBuf[0:0]
	} else {
		scans = make([]plan.Operator, 0, len(indexes))
	}

	sargLength := 0
	var scan plan.Operator
	for index, entry := range indexes {
		if this.order != nil {
			if !this.useIndexOrder(entry, entry.keys) {
				this.resetOrderLimit()
				limit = nil
			} else {
				this.maxParallelism = 1
			}
		}

		arrayIndex := indexHasArrayIndexKey(index)

		if limit != nil {
			var pushDown bool
			if !arrayIndex {
				pushDown, err = allowedPushDown(entry, pred, node.Alias())
				if err != nil {
					return nil, 0, err
				}
			}

			if arrayIndex || !pushDown {
				limit = nil
				this.limit = nil
			}
		}

		if arrayIndex {
			// Array index may include spans to be intersected
			iscans := make([]plan.Operator, 0, len(entry.spans)) // For intersect spans
			spans := make([]*plan.Span, 0, len(entry.spans))     // For non-intersect  spans

			for _, span := range entry.spans {
				if span.Intersect {
					scan = plan.NewIndexScan(index, node, plan.Spans{span}, false, nil, nil, nil)
					scan = plan.NewDistinctScan(scan)
					iscans = append(iscans, scan)
				} else {
					spans = append(spans, span)
				}
			}

			if len(iscans) > 0 {
				this.resetOrderLimit()

				if len(spans) > 0 {
					scan = plan.NewIndexScan(index, node, spans, false, nil, nil, nil)
					scan = plan.NewDistinctScan(scan)
					iscans = append(iscans, scan)
				}

				scans = append(scans, iscans...)
			} else {
				scan = plan.NewIndexScan(index, node, spans, false, nil, nil, nil)
				scan = plan.NewDistinctScan(scan)
				scans = append(scans, scan)
			}
		} else {
			scan = plan.NewIndexScan(index, node, entry.spans, false, limit, nil, nil)

			if len(entry.spans) > 1 && (!entry.exactSpans || pred.MayOverlapSpans()) {
				scan = plan.NewDistinctScan(scan)
			}

			scans = append(scans, scan)
		}

		if len(entry.sargKeys) > sargLength {
			sargLength = len(entry.sargKeys)
		}
	}

	if len(scans) == 1 {
		return scans[0], sargLength, nil
	} else {
		return plan.NewIntersectScan(scans...), sargLength, nil
	}
}

func sargableIndexes(indexes []datastore.Index, pred, subset expression.Expression,
	primaryKey expression.Expressions, formalizer *expression.Formalizer) (
	sargables, all map[datastore.Index]*indexEntry, err error) {

	sargables = make(map[datastore.Index]*indexEntry, len(indexes))
	all = make(map[datastore.Index]*indexEntry, len(indexes))
	var keys expression.Expressions

	for _, index := range indexes {
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
					return nil, nil, err
				}

				dnf := NewDNF(key)
				key, err = dnf.Map(key)
				if err != nil {
					return nil, nil, err
				}

				keys[i] = key
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
				return nil, nil, err
			}

			origCond = cond.Copy()

			dnf := NewDNF(cond)
			cond, err = dnf.Map(cond)
			if err != nil {
				return nil, nil, err
			}

			if !SubsetOf(subset, cond) {
				continue
			}
		}

		n := SargableFor(pred, keys)
		entry := &indexEntry{index, keys, keys[0:n], cond, origCond, nil, false}
		all[index] = entry

		if n > 0 {
			sargables[index] = entry
		}
	}

	return sargables, all, nil
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

	return len(se.sargKeys) > len(te.sargKeys) ||
		(shortest && (len(se.keys) <= len(te.keys)))
}

func sargIndexes(sargables map[datastore.Index]*indexEntry, pred expression.Expression) (
	map[datastore.Index]*indexEntry, error) {
	for _, se := range sargables {
		spans, exactSpans, err := SargFor(pred, se.sargKeys, len(se.keys))
		if err != nil || len(spans) == 0 {
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
	if len(entry.spans) > 1 {
		return false
	}

	var filters map[string]value.Value
	if entry.cond != nil {
		filters = _FILTER_COVERS_POOL.Get()
		defer _FILTER_COVERS_POOL.Put(filters)
		filters = entry.cond.FilterCovers(filters)
	}

	i := 0
	for _, orderTerm := range this.order.Terms() {
		// orderTerm is constant
		if orderTerm.Expression().Static() != nil {
			continue
		}

		// non-constant orderTerms are more than index keys
		if i >= len(keys) {
			// match with condition EQ terms
			if equalConditionFilter(filters, orderTerm.Expression().String()) {
				continue
			}
			return false
		}

		if orderTerm.Descending() {
			return false
		}

		if isArray, _ := entry.keys[i].IsArrayIndexKey(); isArray {
			return false
		}

	loop:
		for {
			if orderTerm.Expression().EquivalentTo(keys[i]) {
				// orderTerm matched with index key
				i++
				break loop
			} else if equalConditionFilter(filters, orderTerm.Expression().String()) {
				// orderTerm matched with Condition EQ
				break loop
			} else if equalRangeKey(i, entry.spans[0].Range.Low, entry.spans[0].Range.High) {
				// orderTerm matched with leading Equal Range key
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

func allowedPushDown(entry *indexEntry, pred expression.Expression, alias string) (bool, error) {
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
