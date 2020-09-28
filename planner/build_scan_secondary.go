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

func (this *builder) buildSecondaryScan(indexes, flex map[datastore.Index]*indexEntry,
	node *algebra.KeyspaceTerm, baseKeyspace *base.BaseKeyspace, id expression.Expression,
	searchSargables []*indexEntry) (scan plan.SecondaryScan, sargLength int, err error) {

	scan, sargLength, err = this.buildCovering(indexes, flex, node, baseKeyspace, id, searchSargables)
	if scan != nil || err != nil {
		return
	}

	if this.group != nil {
		this.resetPushDowns()
	}

	pred := baseKeyspace.DnfPred()

	err = this.sargIndexes(baseKeyspace, node.IsUnderHash(), indexes)
	if err != nil {
		return nil, 0, err
	}

	for _, entry := range indexes {
		entry.pushDownProperty = this.indexPushDownProperty(entry, entry.keys, nil,
			pred, node.Alias(), false, false)
	}

	indexes = this.minimalIndexes(indexes, true, pred, node)
	// Already done. need only for one index
	// flex = this.minimalFTSFlexIndexes(flex, true)
	searchSargables = this.minimalSearchIndexes(flex, searchSargables)

	orderEntry, limit, searchOrders := this.buildSecondaryScanPushdowns(indexes, flex, searchSargables, pred, node)

	// Ordering scan, if any, will go into scans[0]
	var scanBuf [16]plan.SecondaryScan
	var scans []plan.SecondaryScan
	var indexProjection *plan.IndexProjection
	sargLength = 0
	cap := len(indexes) + len(flex) + len(searchSargables)

	if cap <= len(scanBuf) {
		scans = scanBuf[0:1]
	} else {
		scans = make([]plan.SecondaryScan, 1, cap)
	}

	if len(indexes) == 1 {
		for _, entry := range indexes {
			indexProjection = this.buildIndexProjection(entry, nil, nil, true)
			if cap == 1 && this.offset != nil && !entry.IsPushDownProperty(_PUSHDOWN_OFFSET) {
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

		filters := baseKeyspace.Filters()
		if filters != nil {
			filters.ClearPlanFlags()
		}
		scan = entry.spans.CreateScan(index, node, this.context.IndexApiVersion(), false, false,
			pred.MayOverlapSpans(), false, this.offset, this.limit, indexProjection,
			indexKeyOrders, nil, nil, nil, filters, entry.cost, entry.cardinality)

		if orderEntry != nil && index == orderEntry.index {
			scans[0] = scan
		} else {
			scans = append(scans, scan)
		}

		if len(entry.sargKeys) > sargLength {
			sargLength = len(entry.sargKeys)
		}
	}

	this.resetProjection()

	// Search() access path for flex indexes
	for index, entry := range flex {
		sfn := entry.sargKeys[0].(*search.Search)
		sOrders := searchOrders
		if entry != orderEntry {
			sOrders = nil
		}
		scan := this.CreateFTSSearch(index, node, sfn, sOrders, nil, nil)
		if entry == orderEntry {
			scans[0] = scan
		} else {
			scans = append(scans, scan)
		}

		if entry.maxKeys > sargLength {
			sargLength = entry.maxKeys
		}
	}

	for _, entry := range searchSargables {
		sfn := entry.sargKeys[0].(*search.Search)
		sOrders := searchOrders
		if entry != orderEntry {
			sOrders = nil
		}
		scan := this.CreateFTSSearch(entry.index, node, sfn, sOrders, nil, nil)
		if entry == orderEntry {
			scans[0] = scan
		} else {
			scans = append(scans, scan)
		}

		if entry.maxKeys > sargLength {
			sargLength = entry.maxKeys
		}
	}

	if len(scans) == 1 {
		this.orderScan = scans[0]
		return scans[0], sargLength, nil
	} else if scans[0] == nil && len(scans) == 2 {
		return scans[1], sargLength, nil
	} else if scans[0] == nil {
		cost, cardinality := this.intersectScanCost(node, scans[1:]...)
		return plan.NewIntersectScan(limit, cost, cardinality, scans[1:]...), sargLength, nil
	} else {
		cost, cardinality := this.intersectScanCost(node, scans...)
		scan = plan.NewOrderedIntersectScan(nil, cost, cardinality, scans...)
		this.orderScan = scan
		return scan, sargLength, nil
	}
}

func (this *builder) buildSecondaryScanPushdowns(indexes, flex map[datastore.Index]*indexEntry,
	searchSargables []*indexEntry, pred expression.Expression, node *algebra.KeyspaceTerm) (
	orderEntry *indexEntry, limit expression.Expression, searchOrders []string) {

	// get ordered Index. If any index doesn't have Limit/Offset pushdown those must turned off
	pushDown := this.hasOffsetOrLimit()
	for _, entry := range indexes {
		if this.order != nil && orderEntry == nil && entry.IsPushDownProperty(_PUSHDOWN_ORDER) {
			orderEntry = entry
		}

		if pushDown && ((this.offset != nil && !entry.IsPushDownProperty(_PUSHDOWN_OFFSET)) ||
			(this.limit != nil && !entry.IsPushDownProperty(_PUSHDOWN_LIMIT))) {
			pushDown = false
		}
	}

	for _, entry := range flex {
		if this.order != nil && orderEntry == nil && entry.IsPushDownProperty(_PUSHDOWN_ORDER) {
			orderEntry = entry
			searchOrders = entry.searchOrders
		}

		if pushDown && ((this.offset != nil && !entry.IsPushDownProperty(_PUSHDOWN_OFFSET)) ||
			(this.limit != nil && !entry.IsPushDownProperty(_PUSHDOWN_LIMIT))) {
			pushDown = false
		}
	}

	if len(searchSargables) > 0 {
		if !pushDown {
			this.resetOffsetLimit()
		}

		var searchOrderEntry *indexEntry

		searchOrderEntry, searchOrders, _ = this.searchPagination(searchSargables, pred, node.Alias())
		pushDown = this.hasOffsetOrLimit()
		if orderEntry == nil {
			orderEntry = searchOrderEntry
		}
	}

	// ordered index found turn off parallelism. If not turn off pushdowns
	if orderEntry != nil {
		this.maxParallelism = 1
	} else if this.order != nil {
		this.resetOrderOffsetLimit()
		return nil, nil, nil
	}

	// if not single index turn off pushdowns and apply on IntersectScan
	if pushDown && (len(indexes)+len(flex)+len(searchSargables)) > 1 {
		limit = offsetPlusLimit(this.offset, this.limit)
		this.resetOffsetLimit()
	} else if !pushDown {
		this.resetOffsetLimit()
	}
	return
}

func (this *builder) sargableIndexes(indexes []datastore.Index, pred, subset expression.Expression,
	primaryKey expression.Expressions, formalizer *expression.Formalizer,
	ubs expression.Bindings, join bool) (
	sargables, all, arrays, flex map[datastore.Index]*indexEntry, err error) {

	sargables = make(map[datastore.Index]*indexEntry, len(indexes))
	all = make(map[datastore.Index]*indexEntry, len(indexes))
	arrays = make(map[datastore.Index]*indexEntry, len(indexes))

	var keys expression.Expressions
	var entry *indexEntry
	var flexRequest *datastore.FTSFlexRequest

	for _, index := range indexes {
		if index.Type() == datastore.FTS {
			if this.hintIndexes {
				// FTS Flex index sargability
				if flexRequest == nil {
					flex = make(map[datastore.Index]*indexEntry, len(indexes))
					flexRequest = this.buildFTSFlexRequest(formalizer.Keyspace(), pred, ubs)
				}

				entry, err = this.sargableFlexSearchIndex(index, flexRequest, join)
				if err != nil {
					return
				}

				if entry != nil {
					flex[index] = entry
				}
			}
			continue
		}

		var isArray bool

		if index.IsPrimary() {
			if primaryKey != nil {
				keys = primaryKey
			} else {
				continue
			}
		} else {
			keys = expression.CopyExpressions(index.RangeKey())

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

		skip := useSkipIndexKeys(index, this.context.IndexApiVersion())
		min, max, sum := SargableFor(pred, keys, false, skip)

		n := min
		if skip {
			n = max
		}

		entry := newIndexEntry(index, keys, keys[0:n], partitionKeys, min, n, sum, cond, origCond, nil, false)
		all[index] = entry

		if min > 0 {
			sargables[index] = entry
		}

		if isArray {
			arrays[index] = entry
		}
	}

	return sargables, all, arrays, flex, nil
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
	pred expression.Expression, node *algebra.KeyspaceTerm) map[datastore.Index]*indexEntry {

	alias := node.Alias()
	useCBO := this.useCBO

	for s, se := range sargables {
		if this.useCBO && se.cost <= 0.0 {
			cost, selec, card, e := indexScanCost(se.index, se.sargKeys, this.context.RequestId(), se.spans, alias)
			if e != nil || (cost <= 0.0 || card <= 0.0) {
				useCBO = false
			} else {
				se.cost = cost
				se.cardinality = card
				se.selectivity = selec
			}
		}

		for t, te := range sargables {
			if t == s {
				continue
			}

			se_pushdown := se.PushDownProperty()
			te_pushdown := te.PushDownProperty()
			if narrowerOrEquivalent(se, te, shortest, pred) {
				if shortest && narrowerOrEquivalent(te, se, shortest, pred) &&
					(te_pushdown > se_pushdown ||
						(te_pushdown == se_pushdown && len(se.keys) > len(te.keys))) {
					delete(sargables, s)
					break
				}
				delete(sargables, t)
			}
		}
	}

	if useCBO && !shortest && len(sargables) > 1 {
		sargables = this.chooseIntersectScan(sargables, node)
	}

	return sargables
}

/*
Is se narrower or equivalent to te.
*/
func narrowerOrEquivalent(se, te *indexEntry, shortest bool, pred expression.Expression) bool {
	if te.minKeys > se.minKeys {
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
	nkmatch := 0
outer:
	for ti := 0; ti < te.minKeys; ti++ {
		tk := te.sargKeys[ti]
		for si := 0; si < se.minKeys; si++ {
			sk := se.sargKeys[si]
			if SubsetOf(sk, tk) || sk.DependsOn(tk) {
				nkmatch++
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

	if te.minKeys == nfcmatch || (shortest && te.minKeys == (nfcmatch+nkmatch)) {
		return true
	}

	return se.sumKeys > te.sumKeys ||
		(shortest && ((len(se.keys) <= len(te.keys)) ||
			(se.cond != nil && te.cond == nil && se.minKeys == te.minKeys)))
}

func (this *builder) sargIndexes(baseKeyspace *base.BaseKeyspace, underHash bool,
	sargables map[datastore.Index]*indexEntry) error {

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

		useFilters := true
		if isOrPred {
			useFilters = false
		} else {
			for _, key := range se.keys {
				if _, ok := key.(*expression.And); ok {
					useFilters = false
					break
				}
			}
		}

		if useFilters {
			spans, exactSpans, err = SargForFilters(baseKeyspace.Filters(), se.keys,
				se.maxKeys, underHash, this.useCBO, baseKeyspace)
		} else {
			spans, exactSpans, err = SargFor(baseKeyspace.DnfPred(), se.keys,
				se.maxKeys, orIsJoin, this.useCBO, baseKeyspace)
		}
		if err != nil || spans.Size() == 0 {
			logging.Errorp("Sargable index not sarged", logging.Pair{"pred", fmt.Sprintf("<ud>%v</ud>", pred)},
				logging.Pair{"sarg_keys",
					fmt.Sprintf("<ud>%v</ud>", se.sargKeys)}, logging.Pair{"error", err})

			return errors.NewPlanError(nil,
				fmt.Sprintf("Sargable index not sarged; pred=%v, sarg_keys=%v, error=%v",
					pred.String(), se.sargKeys.String(), err))
		}

		se.spans = spans
		if exactSpans && !useIndex2API(se.index, this.context.IndexApiVersion()) {
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

func (this *builder) chooseIntersectScan(sargables map[datastore.Index]*indexEntry,
	node *algebra.KeyspaceTerm) map[datastore.Index]*indexEntry {

	keyspace, err := this.getTermKeyspace(node)
	if err != nil {
		return sargables
	}

	indexes := make(map[datastore.Index]*base.IndexCost, len(sargables))

	var bestIndex datastore.Index
	for s, _ := range sargables {
		indexes[s] = base.NewIndexCost(sargables[s].cost, sargables[s].cardinality, sargables[s].selectivity)
		if bestIndex == nil || sargables[s].PushDownProperty() > sargables[bestIndex].PushDownProperty() {
			bestIndex = s
		}
	}

	// if pushdown can be used on an index, use that index
	if bestIndex != nil && sargables[bestIndex].PushDownProperty() > _PUSHDOWN_NONE {
		for s, _ := range sargables {
			if s != bestIndex {
				delete(sargables, s)
			}
		}

		return sargables
	}

	indexes = optChooseIntersectScan(keyspace, indexes)

	for s, _ := range sargables {
		if _, ok := indexes[s]; !ok {
			delete(sargables, s)
		}
	}

	return sargables
}
