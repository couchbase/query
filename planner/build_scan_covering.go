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
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

// Covering Scan

func (this *builder) buildCovering(indexes, flex map[datastore.Index]*indexEntry,
	node *algebra.KeyspaceTerm, baseKeyspace *base.BaseKeyspace, id expression.Expression,
	searchSargables []*indexEntry) (scan plan.SecondaryScan, sargLength int, err error) {

	// covering turrned off or ANSI NEST
	if this.cover == nil || node.IsAnsiNest() {
		return
	}

	hasDeltaKeyspace := this.context.HasDeltaKeyspace(baseKeyspace.Keyspace())

	// GSI covering scan
	scan, sargLength, err = this.buildCoveringScan(indexes, node, baseKeyspace, id)
	if scan != nil || err != nil {
		return
	}

	// Delta keyspace present no covering
	if hasDeltaKeyspace {
		return
	}

	// Flex FTS covering scan
	scan, sargLength, err = this.buildFlexSearchCovering(flex, node, baseKeyspace, id)
	if scan != nil || err != nil {
		return
	}

	// FTS SEARCH() covering scan
	return this.buildSearchCovering(searchSargables, node, baseKeyspace, id)
}

func (this *builder) buildCoveringScan(idxs map[datastore.Index]*indexEntry,
	node *algebra.KeyspaceTerm, baseKeyspace *base.BaseKeyspace,
	id expression.Expression) (plan.SecondaryScan, int, error) {

	if this.cover == nil || len(idxs) == 0 {
		return nil, 0, nil
	}

	indexes := idxs
	hasDeltaKeyspace := this.context.HasDeltaKeyspace(baseKeyspace.Keyspace())
	if hasDeltaKeyspace {
		indexes = make(map[datastore.Index]*indexEntry, 1)
		for index, entry := range idxs {
			if index.IsPrimary() {
				indexes[index] = entry
				break
			}
		}
		if len(indexes) == 0 {
			return nil, 0, nil
		}
	}

	alias := node.Alias()
	exprs := this.cover.Expressions()
	pred := baseKeyspace.DnfPred()
	origPred := baseKeyspace.OrigPred()

	arrays := _ARRAY_POOL.Get()
	defer _ARRAY_POOL.Put(arrays)

	covering := _COVERING_POOL.Get()
	defer _COVERING_POOL.Put(covering)

	// Remember filter covers
	fc := make(map[datastore.Index]map[*expression.Cover]value.Value, len(indexes))

outer:
	for index, entry := range indexes {
		hasArrayKey := indexHasArrayIndexKey(index)
		if hasArrayKey && (len(arrays) < len(covering)) {
			continue
		}

		// Sarg to set spans
		err := this.sargIndexes(baseKeyspace, node.IsUnderHash(), map[datastore.Index]*indexEntry{index: entry})
		if err != nil {
			return nil, 0, err
		}

		keys := entry.keys

		// Matches execution.spanScan.RunOnce()
		if !index.IsPrimary() {
			keys = append(keys, id)
		}

		// Include filter covers
		coveringExprs, filterCovers, err := indexCoverExpressions(entry, keys, pred, origPred, alias)
		if err != nil {
			return nil, 0, err
		}

		// Skip non-covering index
		for _, expr := range exprs {
			if !expression.IsCovered(expr, alias, coveringExprs) {
				continue outer
			}
		}

		if hasArrayKey {
			arrays[index] = true
		}

		covering[index] = true
		fc[index] = filterCovers
		entry.pushDownProperty = this.indexPushDownProperty(entry, keys, nil, pred, alias, false, covering[index])
	}

	// No covering index available
	if len(covering) == 0 {
		return nil, 0, nil
	}

	useCBO := this.useCBO
	if useCBO {
		for c, _ := range covering {
			entry := indexes[c]
			if entry.cost <= 0.0 {
				cost, _, card, e := indexScanCost(entry.index, entry.sargKeys, this.context.RequestId(),
					entry.spans, node.Alias(), this.advisorValidate(), this.context)
				if e != nil || (cost <= 0.0 || card <= 0.0) {
					useCBO = false
				} else {
					entry.cost = cost
					entry.cardinality = card
				}
			}
		}
	}

	var index datastore.Index
	if useCBO {
		for c, _ := range covering {
			// consider pushdown property before considering cost
			if index == nil {
				index = c
			} else {
				c_pushdown := indexes[c].PushDownProperty()
				i_pushdown := indexes[index].PushDownProperty()
				if (c_pushdown > i_pushdown) ||
					((c_pushdown == i_pushdown) &&
						(indexes[c].cost < indexes[index].cost)) {
					index = c
				}
			}
		}
	} else {
		// Avoid array indexes if possible
		if len(arrays) < len(covering) {
			for a, _ := range arrays {
				delete(covering, a)
			}
		}

	couter:
		// keep indexes with highest continous sargable indexes
		for sc, _ := range covering {
			se := indexes[sc]
			for tc, _ := range covering {
				if sc != tc {
					te := indexes[tc]
					if be := bestIndexBySargableKeys(se, te, se.nEqCond, te.nEqCond); be != nil {
						if be == te {
							delete(covering, sc)
							continue couter
						}
						delete(covering, tc)
					}
				}
			}
		}

		// Keep indexes with max sumKeys
		sumKeys := 0
		for c, _ := range covering {
			if max := indexes[c].sumKeys + indexes[c].nEqCond; max > sumKeys {
				sumKeys = max
			}
		}

		for c, _ := range covering {
			if indexes[c].sumKeys+indexes[c].nEqCond < sumKeys {
				delete(covering, c)
			}
		}

		// Use shortest remaining index
		minLen := 0
		for c, _ := range covering {
			cLen := len(c.RangeKey())
			if index == nil {
				index = c
				minLen = cLen
			} else {
				c_pushdown := indexes[c].PushDownProperty()
				i_pushdown := indexes[index].PushDownProperty()
				if (c_pushdown > i_pushdown) ||
					((c_pushdown == i_pushdown) &&
						(cLen < minLen || (cLen == minLen && c.Condition() != nil))) {
					index = c
					minLen = cLen
				}
			}
		}
	}

	entry := indexes[index]
	sargLength := len(entry.sargKeys)
	keys := entry.keys

	// Matches execution.spanScan.RunOnce()
	if !index.IsPrimary() {
		keys = append(keys, id)
	}

	// Include covering expression from index WHERE clause
	filterCovers := fc[index]

	// Include covering expression from index keys
	covers := make(expression.Covers, 0, len(keys))
	for _, key := range keys {
		covers = append(covers, expression.NewCover(key))
	}

	arrayIndex := arrays[index]
	duplicates := entry.spans.CanHaveDuplicates(index, this.context.IndexApiVersion(), pred.MayOverlapSpans(), false)
	indexProjection := this.buildIndexProjection(entry, exprs, id, index.IsPrimary() || arrayIndex || duplicates)

	// Check and reset pagination pushdows
	indexKeyOrders := this.checkResetPaginations(entry, keys)

	// Build old Aggregates on Index2 only
	scan := this.buildCoveringPushdDownIndexScan2(entry, node, baseKeyspace, pred, indexProjection,
		!arrayIndex, false, covers, filterCovers)
	if scan != nil {
		return scan, sargLength, nil
	}

	// Aggregates check and reset
	var indexGroupAggs *plan.IndexGroupAggregates
	if !entry.IsPushDownProperty(_PUSHDOWN_GROUPAGGS) {
		this.resetIndexGroupAggs()
	}

	// build plan for aggregates
	indexGroupAggs, indexProjection = this.buildIndexGroupAggs(entry, keys, false, indexProjection)
	projDistinct := entry.IsPushDownProperty(_PUSHDOWN_DISTINCT)

	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL
	if useCBO && entry.cost > 0.0 && entry.cardinality > 0.0 {
		if indexGroupAggs != nil {
			cost, cardinality = getIndexGroupAggsCost(index, indexGroupAggs, indexProjection, this.keyspaceNames, entry.cardinality)
			if cost > 0.0 && cardinality > 0.0 {
				entry.cost += cost
				entry.cardinality = cardinality
			}
		} else {
			cost, cardinality = getIndexProjectionCost(index, indexProjection, entry.cardinality)
			if cost > 0.0 && cardinality > 0.0 {
				entry.cost += cost
				entry.cardinality = cardinality
			}
		}
	}

	// generate filters for covering index scan
	var filter expression.Expression
	var err error
	if indexGroupAggs == nil && !hasDeltaKeyspace {
		filter, cost, cardinality, err = this.getIndexFilter(index, node.Alias(), entry.spans,
			covers, filterCovers, entry.cost, entry.cardinality)
		if err != nil {
			return nil, 0, err
		}
		if useCBO {
			entry.cost = cost
			entry.cardinality = cardinality
		}
	}

	// build plan for IndexScan
	scan = entry.spans.CreateScan(index, node, this.context.IndexApiVersion(), false, projDistinct,
		pred.MayOverlapSpans(), false, this.offset, this.limit, indexProjection, indexKeyOrders,
		indexGroupAggs, covers, filterCovers, filter, entry.cost, entry.cardinality, hasDeltaKeyspace)
	if scan != nil {
		if entry.index.Type() != datastore.SYSTEM {
			this.collectIndexKeyspaceNames(baseKeyspace.Keyspace())
		}
		this.coveringScans = append(this.coveringScans, scan)
	}

	return scan, sargLength, nil
}

func (this *builder) checkResetPaginations(entry *indexEntry,
	keys expression.Expressions) (indexKeyOrders plan.IndexKeyOrders) {

	// check order pushdown and reset
	if this.order != nil {
		if entry.IsPushDownProperty(_PUSHDOWN_ORDER) {
			_, indexKeyOrders = this.useIndexOrder(entry, keys)
			this.maxParallelism = 1
		} else {
			this.resetOrderOffsetLimit()
		}
	}

	// check offset push down and convert limit = limit + offset
	if this.offset != nil && !entry.IsPushDownProperty(_PUSHDOWN_OFFSET) {
		this.limit = offsetPlusLimit(this.offset, this.limit)
		this.resetOffset()
	}

	// check limit and reset
	if this.limit != nil && !entry.IsPushDownProperty(_PUSHDOWN_LIMIT) {
		this.resetLimit()
	}
	return
}

func (this *builder) buildCoveringPushdDownIndexScan2(entry *indexEntry, node *algebra.KeyspaceTerm,
	baseKeyspace *base.BaseKeyspace, pred expression.Expression, indexProjection *plan.IndexProjection,
	countPush, array bool, covers expression.Covers, filterCovers map[*expression.Cover]value.Value) plan.SecondaryScan {

	// Aggregates supported pre-Index3
	if (useIndex3API(entry.index, this.context.IndexApiVersion()) &&
		util.IsFeatureEnabled(this.context.FeatureControls(), util.N1QL_GROUPAGG_PUSHDOWN)) || !this.oldAggregates ||
		!entry.IsPushDownProperty(_PUSHDOWN_GROUPAGGS) {
		return nil
	}

	defer func() { this.resetIndexGroupAggs() }()

	var indexKeyOrders plan.IndexKeyOrders

	for _, ag := range this.aggs {
		switch agg := ag.(type) {
		case *algebra.Count:
			if !countPush {
				return nil
			}

			distinct := agg.Distinct()
			op := agg.Operands()[0]
			if !distinct || op.Value() == nil {
				scan := this.buildIndexCountScan(node, entry, pred, distinct, covers, filterCovers)
				this.countScan = scan
				return scan
			}

		case *algebra.Min, *algebra.Max:
			indexKeyOrders = make(plan.IndexKeyOrders, 1)
			if _, ok := agg.(*algebra.Min); ok {
				indexKeyOrders[0] = plan.NewIndexKeyOrders(0, false)
			} else {
				indexKeyOrders[0] = plan.NewIndexKeyOrders(0, true)
			}
		default:
			return nil
		}
	}

	this.maxParallelism = 1
	scan := entry.spans.CreateScan(entry.index, node, this.context.IndexApiVersion(), false, false, pred.MayOverlapSpans(),
		array, nil, expression.ONE_EXPR, indexProjection, indexKeyOrders, nil, covers, filterCovers, nil,
		OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, false)
	if scan != nil {
		if entry.index.Type() != datastore.SYSTEM {
			this.collectIndexKeyspaceNames(baseKeyspace.Keyspace())
		}
		this.coveringScans = append(this.coveringScans, scan)
	}

	return scan
}

func mapFilterCovers(fc map[string]value.Value, keyspace string) (map[*expression.Cover]value.Value, error) {
	if len(fc) == 0 {
		return nil, nil
	}

	rv := make(map[*expression.Cover]value.Value, len(fc))
	for s, v := range fc {
		expr, err := parser.Parse(s)
		if err != nil {
			return nil, err
		}

		expression.MarkKeyspace(keyspace, expr)
		c := expression.NewCover(expr)
		rv[c] = v
	}

	return rv, nil
}

func indexCoverExpressions(entry *indexEntry, keys expression.Expressions, pred, origPred expression.Expression, keyspace string) (
	expression.Expressions, map[*expression.Cover]value.Value, error) {

	var filterCovers map[*expression.Cover]value.Value
	exprs := make(expression.Expressions, 0, len(keys))
	exprs = append(exprs, keys...)
	if entry.cond != nil {
		var err error
		fc := _FILTER_COVERS_POOL.Get()
		defer _FILTER_COVERS_POOL.Put(fc)
		fc = entry.cond.FilterCovers(fc)
		fc = entry.origCond.FilterCovers(fc)
		filterCovers, err = mapFilterCovers(fc, keyspace)
		if err != nil {
			return nil, nil, err
		}
	}

	// Allow array indexes to cover ANY predicates
	if pred != nil && entry.exactSpans && indexHasArrayIndexKey(entry.index) {
		sargKeysHasArray := hasArrayIndexKey(entry.sargKeys)

		if _, ok := entry.spans.(*IntersectSpans); !ok && sargKeysHasArray {
			covers, err := CoversFor(pred, origPred, keys)
			if err != nil {
				return nil, nil, err
			}

			if len(covers) > 0 {
				if len(filterCovers) == 0 {
					filterCovers = covers
				} else {
					for c, v := range covers {
						if _, ok := filterCovers[c]; !ok {
							filterCovers[c] = v
						}
					}
				}
			}
		}
	}

	if len(filterCovers) > 0 {
		exprs = make(expression.Expressions, len(keys), len(keys)+len(filterCovers))
		copy(exprs, keys)

		for c, _ := range filterCovers {
			exprs = append(exprs, c.Covered())
		}
	}

	return exprs, filterCovers, nil
}

var _ARRAY_POOL = datastore.NewIndexBoolPool(64)
var _COVERING_POOL = datastore.NewIndexBoolPool(64)
var _FILTER_COVERS_POOL = value.NewStringValuePool(32)
var _STRING_BOOL_POOL = util.NewStringBoolPool(1024)
