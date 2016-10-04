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
	keys       expression.Expressions
	sargKeys   expression.Expressions
	cond       expression.Expression
	spans      plan.Spans
	exactSpans bool
}

func (this *builder) buildSecondaryScan(indexes map[datastore.Index]*indexEntry,
	node *algebra.KeyspaceTerm, id, pred, limit expression.Expression) (plan.Operator, error) {
	if this.cover != nil {
		scan, err := this.buildCoveringScan(indexes, node, id, pred, limit)
		if scan != nil || err != nil {
			return scan, err
		}
	}

	this.resetCountMin()

	indexes = minimalIndexes(indexes, true)

	var err error
	indexes, err = sargIndexes(indexes, pred)
	if err != nil {
		return nil, err
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

	scans := make([]plan.Operator, 0, len(indexes))
	var op plan.Operator
	for index, entry := range indexes {
		if this.order != nil {
			if !this.useIndexOrder(entry, entry.keys) {
				this.resetOrderLimit()
				limit = nil
			} else {
				this.maxParallelism = 1
			}
		}

		if limit != nil {
			exprs, _, err := indexKeyExpressions(entry, entry.sargKeys)
			if err != nil {
				return nil, err
			}

			if !pred.CoveredBy(node.Alias(), exprs) {
				this.limit = nil
				limit = nil
			}
		}

		arrayIndex := indexHasArrayIndexKey(index)

		if limit != nil && (arrayIndex || !allowedPushDown(entry, pred)) {
			limit = nil
			this.limit = nil
		}

		op = plan.NewIndexScan(index, node, entry.spans, false, limit, nil, nil)

		if arrayIndex || (len(entry.spans) > 1 && (!entry.exactSpans || pred.MayOverlapSpans())) {
			// Use DistinctScan to de-dup array index scans, multiple spans
			op = plan.NewDistinctScan(op)
		}

		scans = append(scans, op)
	}

	if len(scans) > 1 {
		return plan.NewIntersectScan(scans...), nil
	} else {
		return scans[0], nil
	}
}

func sargableIndexes(indexes []datastore.Index, pred, subset expression.Expression,
	primaryKey expression.Expressions, formalizer *expression.Formalizer) (
	sargables, entries map[datastore.Index]*indexEntry, err error) {
	var keys expression.Expressions
	sargables = make(map[datastore.Index]*indexEntry, len(indexes))
	entries = make(map[datastore.Index]*indexEntry, len(indexes))

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
		entry := &indexEntry{keys, keys[0:n], cond, nil, false}
		entries[index] = entry

		if n > 0 {
			sargables[index] = entry
		}
	}

	return sargables, entries, nil
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

outer:
	for _, tk := range te.sargKeys {
		for _, sk := range se.sargKeys {
			if SubsetOf(sk, tk) || sk.DependsOn(tk) {
				continue outer
			}
		}

		return false
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
		filters = entry.cond.FilterCovers(make(map[string]value.Value, 16))
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

func allowedPushDown(entry *indexEntry, pred expression.Expression) bool {
	if !entry.exactSpans {
		return false
	}

	// check for non sargable key is in predicate
	for i := len(entry.sargKeys); i < len(entry.keys); i++ {
		if pred.DependsOn(entry.keys[i]) {
			return false
		}
	}

	return true
}

func indexHasArrayIndexKey(index datastore.Index) bool {
	for _, sk := range index.RangeKey() {
		if isArray, _ := sk.IsArrayIndexKey(); isArray {
			return true
		}
	}
	return false
}
