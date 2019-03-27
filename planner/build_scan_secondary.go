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
	"github.com/couchbase/query/expression/search"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/value"
)

func (this *builder) buildSecondaryScan(indexes map[datastore.Index]*indexEntry,
	node *algebra.KeyspaceTerm, baseKeyspace *base.BaseKeyspace, id expression.Expression,
	searchSargables []*indexEntry) (plan.SecondaryScan, int, error) {

	if this.cover != nil && !node.IsAnsiNest() {
		if len(searchSargables) == 1 {
			scan, sargLength, err := this.buildSearchCoveringScan(searchSargables, node, baseKeyspace, id)
			if scan != nil || err != nil {
				return scan, sargLength, err
			}
		}
		scan, sargLength, err := this.buildCoveringScan(indexes, node, baseKeyspace, id)
		if scan != nil || err != nil {
			return scan, sargLength, err
		}
	}

	this.resetProjection()
	if this.group != nil {
		this.resetPushDowns()
	}

	pred := baseKeyspace.DnfPred()

	err := this.sargIndexes(baseKeyspace, node.IsUnderHash(), indexes)
	if err != nil {
		return nil, 0, err
	}

	for _, entry := range indexes {
		entry.pushDownProperty = this.indexPushDownProperty(entry, entry.keys, nil, pred, node.Alias(), false, false)
	}

	indexes = this.minimalIndexes(indexes, true, pred)

	var orderEntry *indexEntry
	var limit expression.Expression
	pushDown := false

	for _, entry := range indexes {
		if this.order != nil && entry.IsPushDownProperty(_PUSHDOWN_ORDER) {
			orderEntry = entry
			this.maxParallelism = 1
		}

		if !pushDown && len(searchSargables) == 0 &&
			entry.IsPushDownProperty(_PUSHDOWN_LIMIT|_PUSHDOWN_OFFSET) {
			pushDown = true
		}
	}

	searchOrderEntry, searchOrders, _ := this.searchPagination(searchSargables, pred, node.Alias())
	if orderEntry == nil {
		orderEntry = searchOrderEntry
	}
	// No ordering index, disable ORDER and LIMIT pushdown
	if this.order != nil && orderEntry == nil {
		this.resetOrderOffsetLimit()
	}

	if pushDown && len(indexes) > 1 {
		limit = offsetPlusLimit(this.offset, this.limit)
		this.resetOffsetLimit()
	} else if !pushDown && len(searchSargables) == 0 {
		this.resetOffsetLimit()
	}

	// Ordering scan, if any, will go into scans[0]
	var scanBuf [16]plan.SecondaryScan
	var scans []plan.SecondaryScan
	var scan plan.SecondaryScan
	var indexProjection *plan.IndexProjection
	sargLength := 0

	if len(indexes) <= len(scanBuf) {
		scans = scanBuf[0:1]
	} else {
		scans = make([]plan.SecondaryScan, 1, len(indexes))
	}

	if len(indexes) == 1 {
		for _, entry := range indexes {
			indexProjection = this.buildIndexProjection(entry, nil, nil, true)
			if this.offset != nil && !entry.IsPushDownProperty(_PUSHDOWN_OFFSET) {
				this.limit = offsetPlusLimit(this.offset, this.limit)
				this.resetOffset()
			}
			break
		}
	} else {
		indexProjection = this.buildIndexProjection(nil, nil, nil, true)
	}

	for index, entry := range indexes {
		// If this is a join with primary key (meta().id), then it's
		// possible to get right hand documdents directly without
		// accessing through an index (similar to "regular" join).
		// In such cases do not consider secondary indexes that does
		// not include meta().id as a sargable index key. In addition,
		// the index must have either a WHERE clause or at least
		// one other sargable key.
		if node.IsPrimaryJoin() {
			metaFound := false
			for _, key := range entry.sargKeys {
				if key.EquivalentTo(id) {
					metaFound = true
					break
				}
			}

			if !metaFound || (len(entry.sargKeys) <= 1 && index.Condition() == nil) {
				continue
			}
		}

		var indexKeyOrders plan.IndexKeyOrders
		if orderEntry != nil && index == orderEntry.index {
			_, indexKeyOrders = this.useIndexOrder(entry, entry.keys)
		}

		scan = entry.spans.CreateScan(index, node, this.indexApiVersion, false, false, pred.MayOverlapSpans(), false,
			this.offset, this.limit, indexProjection, indexKeyOrders, nil, nil, nil, entry.cost, entry.cardinality)

		if orderEntry != nil && index == orderEntry.index {
			scans[0] = scan
		} else {
			scans = append(scans, scan)
		}

		if len(entry.sargKeys) > sargLength {
			sargLength = len(entry.sargKeys)
		}
	}

	for _, entry := range searchSargables {
		sfn := entry.sargKeys[0].(*search.Search)
		scan := this.CreateFTSSearch(entry.index, node, sfn, searchOrders, nil, nil)
		if entry == orderEntry {
			scans[0] = scan
		} else {
			scans = append(scans, scan)
		}
	}

	if len(scans) == 1 {
		this.orderScan = scans[0]
		return scans[0], sargLength, nil
	} else if scans[0] == nil && len(scans) == 2 {
		return scans[1], sargLength, nil
	} else if scans[0] == nil {
		return plan.NewIntersectScan(limit, scans[1:]...), sargLength, nil
	} else {
		scan = plan.NewOrderedIntersectScan(limit, scans...)
		this.orderScan = scan
		return scan, sargLength, nil
	}
}

func (this *builder) sargableIndexes(indexes []datastore.Index, pred, subset expression.Expression,
	primaryKey expression.Expressions, formalizer *expression.Formalizer) (
	sargables, all, arrays map[datastore.Index]*indexEntry, err error) {

	sargables = make(map[datastore.Index]*indexEntry, len(indexes))
	all = make(map[datastore.Index]*indexEntry, len(indexes))
	arrays = make(map[datastore.Index]*indexEntry, len(indexes))

	var keys expression.Expressions

	for _, index := range indexes {
		var isArray bool

		if index.Type() == datastore.FTS {
			continue
		} else if index.IsPrimary() {
			if primaryKey != nil {
				keys = primaryKey
			} else {
				continue
			}
		} else {
			keys = index.RangeKey()

			if len(keys) == 0 {
				continue
			}

			keys = keys.Copy()

			for i, key := range keys {
				key = key.Copy()

				formalizer.SetIndexScope()
				key, err = formalizer.Map(key)
				formalizer.ClearIndexScope()
				if err != nil {
					return
				}

				dnf := NewDNF(key, true, true)
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

			formalizer.SetIndexScope()
			cond, err = formalizer.Map(cond)
			formalizer.ClearIndexScope()
			if err != nil {
				return
			}

			origCond = cond.Copy()

			dnf := NewDNF(cond, true, true)
			cond, err = dnf.Map(cond)
			if err != nil {
				return
			}

			if !SubsetOf(subset, cond) {
				continue
			}
		}

		var partitionKeys expression.Expressions
		partitionKeys, err = indexPartitionKeys(index, formalizer)
		if err != nil {
			return
		}

		skip := useSkipIndexKeys(index, this.indexApiVersion)
		min, max, sum := SargableFor(pred, keys, false, skip)

		n := min
		if skip {
			n = max
		}

		entry := newIndexEntry(index, keys, keys[0:n], partitionKeys, n, sum, cond, origCond, nil, false)
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

func indexPartitionKeys(index datastore.Index,
	formalizer *expression.Formalizer) (partitionKeys expression.Expressions, err error) {

	index3, ok := index.(datastore.Index3)
	if !ok {
		return
	}

	partitionInfo, _ := index3.PartitionKeys()
	if partitionInfo == nil || partitionInfo.Strategy == datastore.NO_PARTITION {
		return
	}

	partitionKeys = partitionInfo.Exprs
	if formalizer == nil {
		return partitionKeys, err
	}

	partitionKeys = partitionKeys.Copy()
	for i, key := range partitionKeys {
		key = key.Copy()

		partitionKeys[i], err = formalizer.Map(key)
		if err != nil {
			return nil, err
		}
	}
	return partitionKeys, err
}

func (this *builder) minimalIndexes(sargables map[datastore.Index]*indexEntry, shortest bool,
	pred expression.Expression) map[datastore.Index]*indexEntry {

	for s, se := range sargables {
		useCBO := this.useCBO
		if useCBO {
			if se.cost < 0 {
				cost, _, card, e := indexScanCost(se.index, se.sargKeys, this.requestId, se.spans)
				if e != nil {
					useCBO = false
				} else {
					se.cost = cost
					se.cardinality = card
				}
			}
		}

		for t, te := range sargables {
			if t == s {
				continue
			}

			if useCBO {
				if te.cost < 0 {
					cost, _, card, e := indexScanCost(te.index, te.sargKeys, this.requestId, te.spans)
					if e != nil {
						useCBO = false
					} else {
						te.cost = cost
						te.cardinality = card
					}
				}

				// consider pushdown property before considering cost
				se_pushdown := se.PushDownProperty()
				te_pushdown := te.PushDownProperty()
				if se_pushdown > te_pushdown ||
					((se_pushdown == te_pushdown) && (se.cost < te.cost)) {
					delete(sargables, t)
				}
			} else {
				if narrowerOrEquivalent(se, te, shortest, pred) {
					if shortest && narrowerOrEquivalent(te, se, shortest, pred) &&
						te.PushDownProperty() > se.PushDownProperty() {
						delete(sargables, s)
						break
					}
					delete(sargables, t)
				}
			}
		}
	}

	return sargables
}

/*
Is se narrower or equivalent to te.
*/
func narrowerOrEquivalent(se, te *indexEntry, shortest bool, pred expression.Expression) bool {
	if len(te.sargKeys) > len(se.sargKeys) {
		return false
	}

	if te.cond != nil && (se.cond == nil || !SubsetOf(se.cond, te.cond)) {
		return false
	}

	var fc map[string]value.Value
	var predFc map[string]value.Value
	if se.cond != nil {
		fc = _FILTER_COVERS_POOL.Get()
		defer _FILTER_COVERS_POOL.Put(fc)
		fc = se.cond.FilterCovers(fc)
	}

	if shortest && pred != nil {
		predFc = _FILTER_COVERS_POOL.Get()
		defer _FILTER_COVERS_POOL.Put(predFc)
		predFc = pred.FilterCovers(predFc)
	}

	nfcmatch := 0
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

		/* Count number of matches
		 * Indexkey is part of other index condition as equality predicate
		 * If trying to determine shortest index(For: IntersectScan)
		 *     indexkey is not equality predicate and indexkey is part of other index condition
		 *     (In case of equality predicate keeping IntersectScan might be better)
		 */
		_, condEq := fc[tk.String()]
		_, predEq := predFc[tk.String()]
		if condEq || (shortest && !predEq && se.cond.DependsOn(tk)) {
			nfcmatch++
		} else {
			return false
		}
	}

	if len(te.sargKeys) == nfcmatch {
		return true
	}

	return se.sumKeys > te.sumKeys ||
		(shortest && ((len(se.keys) <= len(te.keys)) ||
		(se.cond != nil && te.cond == nil && len(se.sargKeys) == len(te.sargKeys))))
}

func (this *builder) sargIndexes(baseKeyspace *base.BaseKeyspace, underHash bool, sargables map[datastore.Index]*indexEntry) error {

	pred := baseKeyspace.DnfPred()
	isOrPred := false
	orIsJoin := false
	if !underHash {
		if _, ok := pred.(*expression.Or); ok {
			isOrPred = true
			for _, fl := range baseKeyspace.Filters() {
				if fl.IsJoin() {
					orIsJoin = true
					break
				}
			}
		}
	}

	for _, se := range sargables {
		var spans SargSpans
		var exactSpans bool
		var err error

		if isOrPred {
			spans, exactSpans, err = SargFor(baseKeyspace.DnfPred(), se.keys, se.minKeys, orIsJoin, this.useCBO, baseKeyspace)
		} else {
			spans, exactSpans, err = SargForFilters(baseKeyspace.Filters(), se.keys, se.minKeys, underHash, this.useCBO, baseKeyspace)
		}
		if err != nil || spans.Size() == 0 {
			logging.Errorp("Sargable index not sarged", logging.Pair{"pred", fmt.Sprintf("<ud>%v</ud>", pred)},
				logging.Pair{"sarg_keys", fmt.Sprintf("<ud>%v</ud>", se.sargKeys)}, logging.Pair{"error", err})

			return errors.NewPlanError(nil, fmt.Sprintf("Sargable index not sarged; pred=%v, sarg_keys=%v, error=%v",
				pred.String(), se.sargKeys.String(), err))
		}

		se.spans = spans
		if exactSpans && !useIndex2API(se.index, this.indexApiVersion) {
			exactSpans = spans.ExactSpan1(len(se.keys))
		}
		se.exactSpans = exactSpans
	}

	return nil
}

func indexHasArrayIndexKey(index datastore.Index) bool {
	for _, sk := range index.RangeKey() {
		if isArray, _ := sk.IsArrayIndexKey(); isArray {
			return true
		}
	}
	return false
}
