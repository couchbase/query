//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

func (this *builder) buildSecondaryScan(indexes, arrayIndexes, flex map[datastore.Index]*indexEntry,
	node *algebra.KeyspaceTerm, baseKeyspace *base.BaseKeyspace, subset, id expression.Expression,
	searchSargables []*indexEntry) (scan plan.SecondaryScan, sargLength int, err error) {

	pred := baseKeyspace.DnfPred()
	origPred := baseKeyspace.OrigPred()
	unnests, primaryUnnests, unnestIndexes := this.buildUnnestIndexes(node, this.from,
		pred, arrayIndexes)
	defer releaseUnnestPools(unnests, primaryUnnests)

	indexAll := this.hintIndexes && baseKeyspace.HasIndexAllHint()

	if !this.hasBuilderFlag(BUILDER_DO_JOIN_FILTER) && !indexAll {
		scan, sargLength, err = this.buildCovering(indexes, unnestIndexes, flex, node,
			baseKeyspace, subset, id, searchSargables, unnests)
		if scan != nil || err != nil {
			return
		}
	}

	// if considering primary scan for nested-loop join, we can bail out here after
	// consideration of covering scan above
	if this.hasBuilderFlag(BUILDER_JOIN_ON_PRIMARY) {
		return
	}

	hasDeltaKeyspace := this.context.HasDeltaKeyspace(baseKeyspace.Keyspace())

	if this.group != nil || hasDeltaKeyspace {
		this.resetPushDowns()
	}

	for idx, entry := range indexes {
		entry.pushDownProperty = this.indexPushDownProperty(entry, entry.keys, nil,
			pred, origPred, node.Alias(), nil, false, false, (len(this.baseKeyspaces) == 1),
			implicitAnyCover(entry, true, this.context.FeatureControls()))

		err = this.getIndexFilters(entry, node, baseKeyspace, id)
		if err != nil {
			return
		}
		if this.hasBuilderFlag(BUILDER_DO_JOIN_FILTER) && !entry.HasFlag(IE_HAS_JOIN_FILTER) {
			delete(indexes, idx)
		}
	}

	if len(primaryUnnests) > 0 && len(unnests) > 0 && len(unnestIndexes) > 0 {
		var unnestSargables map[datastore.Index]*indexEntry
		unnestSargables, err = this.buildUnnestScan(node, pred, subset, origPred, unnests,
			primaryUnnests, unnestIndexes, hasDeltaKeyspace)
		if err != nil {
			return
		}
		// add to regular Secondary IndexScan list
		for index, entry := range unnestSargables {
			indexes[index] = entry
		}
	}

	return this.buildCreateSecondaryScan(indexes, flex, node, baseKeyspace,
		pred, id, searchSargables, indexAll, hasDeltaKeyspace)
}

func (this *builder) buildCreateSecondaryScan(indexes, flex map[datastore.Index]*indexEntry,
	node *algebra.KeyspaceTerm, baseKeyspace *base.BaseKeyspace, pred, id expression.Expression,
	searchSargables []*indexEntry, indexAll, hasDeltaKeyspace bool) (
	scan plan.SecondaryScan, sargLength int, err error) {

	if indexAll {
		indexes = getIndexAllIndexes(indexes, baseKeyspace)
		if len(indexes) < 2 {
			return nil, 0, errors.NewPlanInternalError(fmt.Sprintf("buildCreateSecondaryScan: unexpected number of indexes "+
				"(%d) for keyspace %s with INDEX_ALL hint", len(indexes), node.Alias()))
		}
	} else {
		indexes = this.minimalIndexes(indexes, true, pred, node)
	}

	if !this.hasBuilderFlag(BUILDER_DO_JOIN_FILTER) {
		// Already done. need only for one index
		// flex = this.minimalFTSFlexIndexes(flex, true)
		searchSargables = this.minimalSearchIndexes(flex, searchSargables)
	}

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
			indexProjection = this.buildIndexProjection(entry, nil, nil, true, nil)
			if cap == 1 && this.offset != nil && !entry.IsPushDownProperty(_PUSHDOWN_OFFSET) &&
				!entry.HasFlag(IE_HAS_EARLY_ORDER) {
				this.limit = offsetPlusLimit(this.offset, this.limit)
				this.resetOffset()
			}
			break
		}
	} else {
		indexProjection = this.buildIndexProjection(nil, nil, nil, true, nil)
	}

	hasVector := false
	hasRerank := false
	for index, entry := range indexes {
		// skip primary index with no sargable keys. Able to do PrimaryScan
		if index.IsPrimary() && entry.minKeys == 0 {
			continue
		}
		// If this is a join with primary key (meta().id), then it's
		// possible to get right hand documdents directly without
		// accessing through an index (similar to lookup join).
		// In such cases do not consider secondary indexes that does
		// not include meta().id as a sargable index key. In addition,
		// the index must have either a WHERE clause or at least
		// one other sargable key.
		if node.IsPrimaryJoin() && id != nil {
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

		covers, filterCovers, filter, idxProj, err := this.buildIndexFilters(entry, baseKeyspace, id, node)
		if err != nil {
			return nil, 0, err
		}
		if len(covers) == 0 {
			idxProj = indexProjection
		}

		var indexKeyOrders plan.IndexKeyOrders
		if (orderEntry != nil && index == orderEntry.index) ||
			(entry.IsPushDownProperty(_PUSHDOWN_ORDER) && entry.HasFlag(IE_VECTOR_RERANK)) {
			_, indexKeyOrders, _ = this.entryUseIndexOrder(entry)
		}

		if index.Type() != datastore.SYSTEM {
			this.collectIndexKeyspaceNames(baseKeyspace.Keyspace())
		}

		var limit, offset expression.Expression
		if entry.IsPushDownProperty(_PUSHDOWN_LIMIT) {
			if entry.HasFlag(IE_VECTOR_RERANK) {
				factor := 0
				if index6, ok := index.(datastore.Index6); ok {
					factor = int(index6.RerankFactor())
				}
				if factor <= 0 {
					factor = plan.RERANK_FACTOR
				}
				limit = expandOffsetLimit(this.offset, this.limit, factor)
			} else {
				limit = this.limit
			}
		}
		if entry.IsPushDownProperty(_PUSHDOWN_OFFSET) {
			offset = this.offset
		}

		vector := entry.HasFlag(IE_VECTOR_KEY_SARGABLE)
		hasVector = hasVector || vector

		var indexKeyNames []string
		var indexPartitionSets plan.IndexPartitionSets
		if vector {
			if entry.HasFlag(IE_VECTOR_RERANK) {
				hasRerank = true
			}
			if index6, ok := entry.index.(datastore.Index6); ok && index6.IsBhive() {
				if filter != nil {
					indexKeyNames, err = getIndexKeyNames(node.Alias(), index, idxProj, false)
					if err != nil {
						return nil, 0, err
					}
				}
				indexPartitionSets, err = this.getIndexPartitionSets(entry.partitionKeys, node, pred, baseKeyspace)
				if err != nil {
					return nil, 0, err
				}
			}
		}

		skipNewKeys := false
		if index.Type() == datastore.SEQ_SCAN {
			skipNewKeys = this.skipKeyspace != "" && baseKeyspace.Keyspace() == this.skipKeyspace
			if skipNewKeys {
				this.mustSkipKeys = true
			}
		}

		var includeSpans plan.Spans2
		if tspan, ok := entry.includeSpans.(*TermSpans); ok {
			includeSpans = tspan.spans
		}
		scan = entry.spans.CreateScan(index, node, this.context.IndexApiVersion(), false, false,
			overlapSpans(pred), false, offset, limit, idxProj, indexKeyOrders, nil,
			covers, filterCovers, filter, entry.cost, entry.cardinality,
			entry.size, entry.frCost, includeSpans, baseKeyspace, hasDeltaKeyspace, skipNewKeys,
			this.hasBuilderFlag(BUILDER_NL_INNER), false, indexKeyNames, indexPartitionSets)

		if iscan3, ok := scan.(*plan.IndexScan3); ok {
			if entry.HasFlag(IE_HAS_EARLY_ORDER) {
				iscan3.SetEarlyOrder()
				iscan3.SetEarlyOrderExprs(entry.orderExprs)
				if entry.IsPushDownProperty(_PUSHDOWN_EXACTSPANS) {
					iscan3.SetEarlyLimit()
					if this.offset != nil {
						iscan3.SetEarlyOffset()
					}
				}
			} else if entry.IsPushDownProperty(_PUSHDOWN_ORDER) && !entry.HasFlag(IE_VECTOR_RERANK) &&
				entry.IsPushDownProperty(_PUSHDOWN_EXACTSPANS) {
				if this.limit != nil && !entry.IsPushDownProperty(_PUSHDOWN_LIMIT) {
					iscan3.SetEarlyLimit()
				}
				if this.offset != nil && !entry.IsPushDownProperty(_PUSHDOWN_OFFSET) {
					iscan3.SetEarlyOffset()
				}
			}
		}

		if orderEntry != nil && index == orderEntry.index {
			scans[0] = scan
		} else {
			scans = append(scans, scan)
		}

		if len(entry.sargKeys) > sargLength {
			sargLength = len(entry.sargKeys)
		}
		for _, a := range entry.unnestAliases {
			baseKeyspace.AddUnnestIndex(index, a)
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

		this.collectIndexKeyspaceNames(baseKeyspace.Keyspace())
		scan := this.CreateFTSSearch(index, node, sfn, sOrders, nil, nil, hasDeltaKeyspace, entry.HasFlag(IE_SEARCH_KNN))
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
		if entry.HasFlag(IE_SEARCH_KNN) {
			indexAll = true
		}
		sOrders := searchOrders
		if entry != orderEntry {
			sOrders = nil
		}

		this.collectIndexKeyspaceNames(baseKeyspace.Keyspace())
		scan := this.CreateFTSSearch(entry.index, node, sfn, sOrders, nil, nil,
			hasDeltaKeyspace, entry.HasFlag(IE_SEARCH_KNN))
		if entry == orderEntry {
			scans[0] = scan
		} else {
			scans = append(scans, scan)
		}

		if entry.maxKeys > sargLength {
			sargLength = entry.maxKeys
		}
	}

	if hasVector {
		if !(len(scans) == 1 || (scans[0] == nil && len(scans) == 2)) {
			return nil, 0, errors.NewPlanInternalError(fmt.Sprintf("buildCreateSecondaryScan: vector query using intersect scan (%d)", len(scans)-1))
		}
		if hasRerank {
			this.setBuilderFlag(BUILDER_HAS_VECTOR_RERANK)
		}
	}

	if len(scans) == 1 {
		this.orderScan = scans[0]
		return scans[0], sargLength, nil
	} else if scans[0] == nil && len(scans) == 2 {
		return scans[1], sargLength, nil
	} else if scans[0] == nil {
		cost, cardinality, size, frCost := this.intersectScanCost(node, scans[1:]...)
		return plan.NewIntersectScan(limit, indexAll, cost, cardinality, size, frCost, scans[1:]...), sargLength, nil
	} else {
		cost, cardinality, size, frCost := this.intersectScanCost(node, scans...)
		scan = plan.NewOrderedIntersectScan(nil, indexAll, cost, cardinality, size, frCost, scans...)
		this.orderScan = scan
		return scan, sargLength, nil
	}
}

func (this *builder) buildSecondaryScanPushdowns(indexes, flex map[datastore.Index]*indexEntry,
	searchSargables []*indexEntry, pred expression.Expression, node *algebra.KeyspaceTerm) (
	orderEntry *indexEntry, limit expression.Expression, searchOrders []string) {

	if this.hasBuilderFlag(BUILDER_DO_JOIN_FILTER) {
		return
	}

	hasEarlyOrder := false
	hasRerank := false

	// get ordered Index. If any index doesn't have Limit/Offset pushdown those must turned off
	pushDown := this.hasOffsetOrLimit()
	for _, entry := range indexes {
		if this.order != nil {
			// prefer _PUSHDOWN_ORDER over _PUSHDOWN_PARTIAL_ORDER
			if (orderEntry == nil || orderEntry.IsPushDownProperty(_PUSHDOWN_PARTIAL_ORDER)) &&
				entry.IsPushDownProperty(_PUSHDOWN_ORDER) && !entry.HasFlag(IE_VECTOR_RERANK) {
				orderEntry = entry
				this.partialSortTermCount = 0
			} else if orderEntry == nil && entry.IsPushDownProperty(_PUSHDOWN_PARTIAL_ORDER) {
				orderEntry = entry
				this.partialSortTermCount = entry.partialSortTermCount
			}
		}

		if pushDown && ((this.offset != nil && !entry.IsPushDownProperty(_PUSHDOWN_OFFSET)) ||
			(this.limit != nil && !entry.IsPushDownProperty(_PUSHDOWN_LIMIT))) {
			pushDown = false
		}

		if entry.HasFlag(IE_HAS_EARLY_ORDER) {
			hasEarlyOrder = true
		}

		if entry.HasFlag(IE_VECTOR_RERANK) {
			hasRerank = true
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
		if !pushDown && !hasEarlyOrder {
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
		if !hasEarlyOrder && !hasRerank {
			this.resetOrderOffsetLimit()
		}
		return nil, nil, nil
	}

	// if not single index turn off pushdowns and apply on IntersectScan
	if pushDown && (len(indexes)+len(flex)+len(searchSargables)) > 1 {
		limit = offsetPlusLimit(this.offset, this.limit)
		this.resetOffsetLimit()
	} else if !pushDown && !hasEarlyOrder && !hasRerank {
		this.resetOffsetLimit()
	}
	return
}

func (this *builder) sargableIndexes(indexes []datastore.Index, pred, subset, vpred expression.Expression,
	primaryKey expression.Expressions, formalizer *expression.Formalizer,
	ubs expression.Bindings, join bool, baseKeyspace *base.BaseKeyspace) (
	sargables, arrays, flex map[datastore.Index]*indexEntry, err error) {

	flexPred := pred
	flexVecPred := vpred
	if len(this.context.NamedArgs()) > 0 || len(this.context.PositionalArgs()) > 0 {
		flexPred, err = base.ReplaceParameters(flexPred, this.context.NamedArgs(), this.context.PositionalArgs())
		if err == nil {
			flexVecPred, err = base.ReplaceParameters(flexVecPred, this.context.NamedArgs(), this.context.PositionalArgs())
		}
		if err != nil {
			return
		}
	}

	sargables = make(map[datastore.Index]*indexEntry, len(indexes))
	arrays = make(map[datastore.Index]*indexEntry, len(indexes))

	var keys datastore.IndexKeys
	var entry *indexEntry
	var flexRequest *datastore.FTSFlexRequest
	nvectors := 0

	for _, index := range indexes {
		if index.Type() == datastore.FTS {
			if this.hintIndexes && (flexPred != nil || flexVecPred != nil) && !this.hasBuilderFlag(BUILDER_DO_JOIN_FILTER) {
				// FTS Flex index sargability
				if flexRequest == nil {
					flex = make(map[datastore.Index]*indexEntry, len(indexes))
					flexRequest = this.buildFTSFlexRequest(formalizer.Keyspace(), flexPred, flexVecPred, ubs)
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

		var includes expression.Expressions
		var cond, origCond expression.Expression
		var allKey *expression.All
		var apos int
		var vpos int = -1

		if index.IsPrimary() {
			if primaryKey != nil {
				keys = make(datastore.IndexKeys, 0, len(primaryKey))
				for _, k := range primaryKey {
					keys = append(keys, &datastore.IndexKey{k, datastore.IK_NONE})
				}
			} else {
				continue
			}
		} else {
			cond = index.Condition()
			if cond != nil {
				if subset == nil {
					continue
				}

				if cond, origCond, err = formalizeExpr(formalizer, cond, true); err != nil {
					return
				}

				indexId := index.Id()
				if baseKeyspace.IsNonSubsetIndex(indexId) {
					continue
				} else if !baseKeyspace.IsSubsetIndex(indexId) {
					if !base.SubsetOf(subset, cond) && (origCond == nil || !base.SubsetOf(subset, origCond)) {
						baseKeyspace.AddNonSubsetIndex(indexId)
						continue
					}
					baseKeyspace.AddSubsetIndex(indexId)
				}
			}

			rangeKeys := index.RangeKey()
			for i, key := range rangeKeys {
				if a, ok := key.(*expression.All); ok {
					allKey = a
					apos = i
				}
			}

			keys = datastore.GetIndexKeys(index)

			for i, key := range keys {
				if key.Expr, _, err = formalizeExpr(formalizer, key.Expr, false); err != nil {
					return
				}
				if key.HasAttribute(datastore.IK_VECTOR) {
					vpos = i
				}
			}

			includes = datastore.GetIndexIncludes(index)
			for i, _ := range includes {
				if includes[i], _, err = formalizeExpr(formalizer, includes[i], false); err != nil {
					return
				}
			}
		}

		var partitionKeys expression.Expressions
		partitionKeys, err = indexPartitionKeys(index, formalizer)
		if err != nil {
			return
		}

		skip := useSkipIndexKeys(index, this.context.IndexApiVersion())
		missing := indexHasLeadingKeyMissingValues(index, this.context.FeatureControls())
		min, max, sum, include, skeys := SargableFor(pred, vpred, index, keys, includes,
			missing, skip, nil, this.context, this.aliases)
		exact := min == 0 && pred == nil

		n := min
		if skip && (n > 0 || missing) {
			n = max
		}

		if n == 0 && missing && primaryKey != nil {
			n = 1
		}

		if n > 0 || allKey != nil {
			entry := newIndexEntry(index, keys, includes, n, partitionKeys, min, n, sum,
				include, cond, origCond, nil, exact, nil, false, skeys)
			if missing {
				entry.SetFlags(IE_LEADINGMISSING, true)
			}
			if allKey != nil {
				arrays[index] = entry
				var ak expression.Expression
				if ak, _, err = formalizeExpr(formalizer, allKey, false); err != nil {
					return
				}
				allKey, _ := ak.(*expression.All)
				entry.setArrayKey(allKey, apos)
			}
			if vpos >= 0 && vpos < len(skeys) && skeys[vpos] {
				nvectors++
				entry.SetFlags(IE_VECTOR_KEY_SARGABLE, true)
			}
			if hasRedundantSargKeys(keys, skeys) {
				entry.SetFlags(IE_HAS_REDUNDANT_SARG_KEY, true)
			}

			if n > 0 {
				sargables[index] = entry
			}
		}
	}

	if nvectors > 0 {
		// only keep entries that have vector index key sargable
		for i, e := range sargables {
			if !e.HasFlag(IE_VECTOR_KEY_SARGABLE) {
				delete(sargables, i)
			}
		}
	}

	return sargables, arrays, flex, nil
}

func formalizeExpr(formalizer *expression.Formalizer, expr expression.Expression, orig bool) (
	newExpr, origExpr expression.Expression, err error) {
	if expr != nil {
		expr = expr.Copy()
		formalizer.SetIndexScope()
		expr, err = formalizer.Map(expr)
		formalizer.ClearIndexScope()
		if err == nil {
			if orig {
				origExpr = expr.Copy()
			}

			dnf := base.NewDNF(expr, true, true)
			expr, err = dnf.Map(expr)
		}
	}
	return expr, origExpr, err
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

		formalizer.SetIndexScope()
		partitionKeys[i], err = formalizer.Map(key)
		formalizer.ClearIndexScope()
		if err != nil {
			return nil, err
		}
	}
	return partitionKeys, err
}

func (this *builder) minimalIndexes(sargables map[datastore.Index]*indexEntry, shortest bool,
	pred expression.Expression, node *algebra.KeyspaceTerm) map[datastore.Index]*indexEntry {

	alias := node.Alias()
	baseKeyspace, _ := this.baseKeyspaces[node.Alias()]
	useCBO := this.useCBO && this.keyspaceUseCBO(node.Alias())

	vector := false
	if len(baseKeyspace.VectorFilters()) > 0 {
		for _, entry := range sargables {
			if entry.HasFlag(IE_VECTOR_KEY_SARGABLE) {
				vector = true
				break
			}
		}
	}

	var predFc map[string]value.Value
	if pred != nil {
		predFc = _FILTER_COVERS_POOL.Get()
		defer _FILTER_COVERS_POOL.Put(predFc)
		predFc = pred.FilterCovers(predFc)
	}

	if useCBO {
		advisorValidate := this.advisorValidate()
		keyspace := baseKeyspace.Keyspace()
		for _, se := range sargables {
			// limit_cost indicates whether cost needs to be recalculated due to LIMIT pushdown
			limit_cost := se.IsPushDownProperty(_PUSHDOWN_LIMIT|_PUSHDOWN_OFFSET) &&
				!se.HasFlag(IE_LIMIT_OFFSET_COST) && this.limit != nil
			if se.cost <= 0.0 || limit_cost {
				var limit, offset int64
				if limit_cost {
					limit, offset = this.getLimitOffset(se, this.limit, this.offset)
				}
				cost, selec, card, size, frCost, e := indexScanCost(se, se.sargKeys,
					se.sargIncludes, this.context.RequestId(), se.spans, se.includeSpans,
					alias, baseKeyspace.Keyspace(), limit, offset, advisorValidate, this.context)
				if e != nil || (cost <= 0.0 || card <= 0.0 || size <= 0 || frCost <= 0.0) {
					useCBO = false
				} else {
					se.cardinality, se.selectivity, se.cost, se.frCost, se.size = card, selec, cost, frCost, size
				}
				if limit_cost {
					se.SetFlags(IE_LIMIT_OFFSET_COST, true)
				}
			}
			if shortest && se.fetchCost <= 0.0 {
				fetchCost, _, _ := getFetchCost(keyspace, se.cardinality)
				if fetchCost > 0.0 {
					se.fetchCost = fetchCost
				} else {
					useCBO = false
				}
			}
		}
	}

	if !useCBO && this.hasBuilderFlag(BUILDER_DO_JOIN_FILTER) {
		return nil
	}

	corrSubq := node.IsInCorrSubq()
	for s, se := range sargables {
		for t, te := range sargables {
			if t == s {
				continue
			}

			if useCBO && shortest {
				if t.Type() == datastore.SEQ_SCAN && te.nSargKeys == 0 {
					delete(sargables, t)
					continue
				} else if s.Type() == datastore.SEQ_SCAN && se.nSargKeys == 0 {
					continue
				} else if corrSubq {
					// if inside correlated subquery, skip primary index with
					// no sargable keys (primary scan)
					if t.IsPrimary() && te.minKeys == 0 {
						delete(sargables, t)
						continue
					} else if s.IsPrimary() && se.minKeys == 0 {
						continue
					}
				}

				if matchedLeadingKeys(se, te, predFc) || vectorIndexes(s, t) {
					seCost := se.scanCost()
					teCost := te.scanCost()
					if seCost < teCost || (seCost == teCost && se.cardinality < te.cardinality) {
						delete(sargables, t)
					}
				}
			} else {
				if narrowerOrEquivalent(se, te, shortest, corrSubq, predFc) {
					delete(sargables, t)
				}
			}
		}
	}

	if shortest && len(sargables) > 1 {
		if !useCBO && this.useCBO && this.keyspaceUseCBO(node.Alias()) {
			// reset useCBO
			// it is possible that useCBO is set to false above due to cost not available
			// for one of the indexes, however if that index is eliminated by other
			// criteria already then we may be able to turn useCBO back on again
			useCBO = true
			for _, se := range sargables {
				if se.cost <= 0.0 || se.cardinality <= 0.0 || se.size <= 0 || se.frCost <= 0.0 {
					useCBO = false
					break
				}
			}
		}
		if useCBO {
			sargables = this.chooseIntersectScan(sargables, node, vector)
		}
		if len(sargables) > 1 {
			// remove any early order indicator
			for _, entry := range sargables {
				entry.UnsetFlags(IE_HAS_EARLY_ORDER)
			}
			if vector {
				// choose random one
				for i, e := range sargables {
					return map[datastore.Index]*indexEntry{
						i: e,
					}
				}
			}
		}
	}

	return sargables
}

/*
Is se narrower or equivalent to te.

	true : purge te
	false: keep both
*/
func narrowerOrEquivalent(se, te *indexEntry, shortest, corrSubq bool, predFc map[string]value.Value) bool {

	snk, snc := matchedKeysConditions(se, te, shortest, predFc)

	be := bestIndexBySargableKeys(se, te, se.nEqCond, te.nEqCond, shortest)
	if be == te { // te is better index
		return false
	}

	if te.nSargKeys > 0 && te.nSargKeys == snk+snc && se.nSargKeys > snk && be == se {
		// all te sargable keys are se sargable keys and part of se condition;
		//  se have more different sragable keys
		if !shortest && te.cond != nil && (se.cond == nil || !base.SubsetOf(se.cond, te.cond)) {
			return false
		}
		return true
	}

	// non shortest case keep both indexes so that covering can be done on best index
	if !shortest {
		// prefer an index over a sequential scan
		if te.index.Type() == datastore.SEQ_SCAN {
			return true
		}
		return false
	}

	if be == se {
		return true
	}

	vector := vectorIndexes(se.index, te.index)

	sePushDown := se.PushDownProperty()
	tePushDown := te.PushDownProperty()
	seKeyFlags := se.IndexKeyFlags()
	teKeyFlags := te.IndexKeyFlags()

	if te.nSargKeys == (snk+snc) && sePushDown == tePushDown && seKeyFlags == teKeyFlags {
		// if te and se has same sargKeys (or equivalent condition), and there exists
		// a non-sarged array key, prefer the one without the array key
		if se.HasFlag(IE_ARRAYINDEXKEY) != te.HasFlag(IE_ARRAYINDEXKEY) {
			if !se.HasFlag(IE_ARRAYINDEXKEY_SARGABLE) && !te.HasFlag(IE_ARRAYINDEXKEY_SARGABLE) {
				if te.HasFlag(IE_ARRAYINDEXKEY) && !se.HasFlag(IE_ARRAYINDEXKEY) {
					return true
				} else if se.HasFlag(IE_ARRAYINDEXKEY) && !te.HasFlag(IE_ARRAYINDEXKEY) {
					return false
				}
			}

		}
	}

	if te.cond != nil && (se.cond == nil || !base.SubsetOf(se.cond, te.cond)) {
		return false
	}

	if te.nSargKeys > 0 {
		if te.nSargKeys > (snk + snc) {
			return false
		} else if te.nSargKeys == (snk+snc) && sePushDown == tePushDown && seKeyFlags == teKeyFlags {
			seMinKeys := se.minKeys
			if seMinKeys == len(se.idxKeys) {
				seMinKeys += se.includeKeys
			}
			teMinKeys := te.minKeys
			if teMinKeys == len(te.idxKeys) {
				teMinKeys += te.includeKeys
			}
			if seMinKeys != teMinKeys {
				if seMinKeys != se.nSargKeys || teMinKeys != te.nSargKeys {
					// for two indexes with the same sargKeys, favor the one
					// with more consecutive leading sargKeys
					// e.g (c1, c4) vs (c1, c2, c4) with predicates on c1 and c4
					return seMinKeys > teMinKeys
				} else if (seMinKeys + snc) != teMinKeys {
					// also consider matched index conditions
					return (seMinKeys + snc) > teMinKeys
				}
			}
			seMaxKeys := se.maxKeys + se.includeKeys
			teMaxKeys := te.maxKeys + te.includeKeys
			if seMaxKeys != teMaxKeys {
				if seMaxKeys != se.nSargKeys || teMaxKeys != te.nSargKeys {
					// favor the one with shorter sargKeys
					return seMaxKeys < teMaxKeys
				} else if (seMaxKeys + snc) != teMaxKeys {
					// also consider matched index conditions
					return (seMaxKeys + snc) < teMaxKeys
				}
			}
		}
	}

	seSumKeys := se.sumKeys + se.includeKeys
	teSumKeys := te.sumKeys + te.includeKeys
	if seSumKeys+se.nEqCond != teSumKeys+te.nEqCond {
		return seSumKeys+se.nEqCond > teSumKeys+te.nEqCond
	}

	if sePushDown != tePushDown {
		return sePushDown > tePushDown
	}

	if vector {
		seRerank := se.HasFlag(IE_VECTOR_RERANK)
		teRerank := te.HasFlag(IE_VECTOR_RERANK)
		if seRerank != teRerank {
			return teRerank
		}
	}

	if se.includeKeys != te.includeKeys {
		return se.includeKeys > te.includeKeys
	}

	// prefer one with index filter/join filter/early order
	if seKeyFlags != teKeyFlags {
		return seKeyFlags > teKeyFlags
	}
	seFltr := se.HasFlag(IE_HAS_FILTER)
	teFltr := te.HasFlag(IE_HAS_FILTER)
	if seFltr != teFltr {
		return seFltr
	}

	teType := te.index.Type()
	seType := se.index.Type()
	// prefer an index over a sequential scan
	if teType == datastore.SEQ_SCAN {
		return true
	} else if seType == datastore.SEQ_SCAN {
		return false
	}

	if (se.nCondKeys+snc) != te.nCondKeys && !se.HasFlag(IE_NONEQ_COND) && !te.HasFlag(IE_NONEQ_COND) {
		if (se.nCondKeys + snc) > te.nCondKeys {
			return true
		} else if se.cond == nil || (te.cond != nil && base.SubsetOf(te.cond, se.cond)) {
			return false
		}

	}

	if len(se.keys) != len(te.keys) {
		return len(se.keys) < len(te.keys)
	}

	// for equivalent keys, prefer non-VIRTUAL
	if teType == datastore.VIRTUAL && seType != datastore.VIRTUAL {
		return true
	} else if seType == datastore.VIRTUAL && teType != datastore.VIRTUAL {
		return false
	}

	// when neither index has sargable keys:
	//   - if inside correlated subquery, skip primary index
	//   - otherwise favor primary index over missing-leading key index
	if te.nSargKeys == 0 && se.nSargKeys == 0 {
		if se.index.IsPrimary() {
			if corrSubq {
				return false
			} else if te.cond == nil {
				return true
			}
		} else if te.index.IsPrimary() {
			if corrSubq {
				return true
			} else if se.cond == nil {
				return false
			}
		}
	}

	return se.cond != nil || (te.nSargKeys == (snk + snc))
}

// Calculates how many keys te sargable keys matched with se sargable keys and se condition
// for include columns, if an include column is sargable, consider it the same as index key
func matchedKeysConditions(se, te *indexEntry, shortest bool, predFc map[string]value.Value) (nk, nc int) {

	var sk, tk expression.Expression

outer:
	for ti := 0; ti < len(te.keys)+len(te.includes); ti++ {
		if !te.skeys[ti] {
			continue
		}
		if ti < len(te.sargKeys) {
			tk = te.sargKeys[ti]
		} else if ti >= len(te.keys) {
			tk = te.includes[ti-len(te.keys)]
		} else {
			// not expected
			continue
		}

		for si := 0; si < len(se.keys)+len(se.includes); si++ {
			if se.skeys[si] {
				if si < len(se.sargKeys) {
					sk = se.sargKeys[si]
				} else if si >= len(se.keys) {
					sk = se.includes[si-len(se.keys)]
				} else {
					// not expected
					continue
				}
				if matchedIndexKey(sk, tk) {
					nk++
					continue outer
				}
			}
		}

		if matchedIndexCondition(se, tk, predFc, shortest) {
			nc++
		}
	}

	return
}

// for CBO, prune indexes that has similar leading index keys
func matchedLeadingKeys(se, te *indexEntry, predFc map[string]value.Value) bool {
	if se.nSargKeys == 0 && te.nSargKeys == 0 &&
		se.HasFlag(IE_LEADINGMISSING) && te.HasFlag(IE_LEADINGMISSING) {
		return true
	}

	if se.PushDownProperty() != te.PushDownProperty() ||
		se.IndexKeyFlags() != te.IndexKeyFlags() {
		return false
	}

	nkeys := 0
	ncond := 0
	for i, tk := range te.sargKeys {
		if !te.skeys[i] {
			continue
		}

		if i >= len(se.sargKeys) {
			break
		}

		if !se.skeys[i] {
			continue
		}

		sk := se.sargKeys[i]
		if matchedIndexKey(sk, tk) {
			nkeys++
		} else if matchedIndexCondition(se, tk, predFc, true) {
			ncond++
		} else {
			break
		}
	}

	return (nkeys + ncond) > 0
}

func matchedIndexKey(sk, tk expression.Expression) bool {
	return base.SubsetOf(sk, tk) || sk.DependsOn(tk)
}

func matchedIndexCondition(se *indexEntry, tk expression.Expression, predFc map[string]value.Value,
	shortest bool) bool {

	/* Count number of matches
	 * Indexkey is part of other index condition as equality predicate
	 * If trying to determine shortest index(For: IntersectScan)
	 *     indexkey is not equality predicate and indexkey is part of other index condition
	 *     (In case of equality predicate keeping IntersectScan might be better)
	 */
	_, condEq := se.condFc[tk.String()]
	_, predEq := predFc[tk.String()]
	if condEq || (shortest && !predEq && se.cond != nil && se.cond.DependsOn(tk)) {
		return true
	}

	return false
}

func vectorIndexes(s, t datastore.Index) bool {
	if ixs, ok := s.(datastore.Index6); ok && ixs.IsVector() {
		if ixt, ok := t.(datastore.Index6); ok && ixt.IsVector() {
			return true
		}
	}
	return false
}

func (this *builder) sargIndexes(baseKeyspace *base.BaseKeyspace, underHash bool,
	sargables map[datastore.Index]*indexEntry) error {

	pred := baseKeyspace.DnfPred()
	var vpred expression.Expression
	if !this.hasBuilderFlag(BUILDER_NL_INNER) {
		vpred = baseKeyspace.GetVectorPred()
	}
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

	useCBO := this.useCBO && (baseKeyspace.DocCount() >= 0)
	advisorValidate := this.advisorValidate()
	for _, se := range sargables {
		var spans, includeSpans SargSpans
		var exactSpans, exactIncludes bool
		var err error

		if (se.index.IsPrimary() && se.minKeys == 0) || se.maxKeys == 0 {
			se.spans = _WHOLE_SPANS.Copy()
			if pred != nil {
				se.exactSpans = false
			}
			continue
		}

		useFilters := true
		if isOrPred {
			if this.hasBuilderFlag(BUILDER_OR_SUBTERM) {
				useFilters = false
			} else {
				useFilters = this.orSargUseFilters(pred.(*expression.Or), vpred, baseKeyspace, se)
			}
		} else {
			for _, key := range se.keys {
				if _, ok := key.(*expression.And); ok {
					useFilters = false
					break
				}
			}
		}
		if useFilters && vpred != nil && len(baseKeyspace.Filters()) == 0 {
			useFilters = false
		}
		isMissing := se.HasFlag(IE_LEADINGMISSING)
		validSpans := false
		if useFilters {
			filters := baseKeyspace.Filters()
			if se.exactFilters != nil {
				// already considered before, clear the map
				for k, _ := range se.exactFilters {
					delete(se.exactFilters, k)
				}
			} else {
				se.exactFilters = make(map[*base.Filter]bool, len(filters))
			}
			if se.exactIncludeFilters != nil {
				// already considered before, clear the map
				for k, _ := range se.exactIncludeFilters {
					delete(se.exactIncludeFilters, k)
				}
			} else {
				se.exactIncludeFilters = make(map[*base.Filter]bool, len(filters))
			}
			spans, exactSpans, includeSpans, exactIncludes, err = SargForFilters(filters, vpred,
				se, se.idxKeys, isMissing, nil, se.maxKeys, underHash, useCBO, baseKeyspace,
				this.keyspaceNames, advisorValidate, this.aliases, se.exactFilters,
				se.exactIncludeFilters, this.context)
			if err == nil && (spans != nil || !isOrPred || !se.HasFlag(IE_LEADINGMISSING)) {
				// If this is OR predicate and no valid span generated, and index
				// has leading missing, allow it to try with SargFor() below.
				validSpans = true
			}
		}
		if !validSpans {
			useFilters = false
			spans, exactSpans, includeSpans, exactIncludes, err = SargFor(baseKeyspace.DnfPred(), vpred,
				se, se.idxKeys, isMissing, nil, se.maxKeys, orIsJoin, useCBO, baseKeyspace,
				this.keyspaceNames, advisorValidate, this.aliases, this.context)
		}

		if se.HasFlag(IE_LEADINGMISSING) && (spans == nil || spans.Size() == 0) {
			se.spans = _WHOLE_SPANS.Copy()
			if pred == nil || (se.cond != nil && pred.EquivalentTo(se.cond)) ||
				(se.origCond != nil && pred.EquivalentTo(se.origCond)) {

				se.exactSpans = true
			} else {
				se.exactSpans = false
			}
			continue
		}

		if err != nil || spans == nil || spans.Size() == 0 {
			logging.Errora(func() string {
				if vpred != nil {
					return fmt.Sprintf("Sargable index not sarged: pred:<ud>%v</ud> vector_pred:<ud>%v</ud> sarg_keys:<ud>%v</ud> error:%v",
						pred, vpred, se.sargKeys, err)
				} else {
					return fmt.Sprintf("Sargable index not sarged: pred:<ud>%v</ud> sarg_keys:<ud>%v</ud> error:%v",
						pred, se.sargKeys, err)
				}
			})

			var errMsg string
			if vpred != nil {
				errMsg = fmt.Sprintf("Sargable index not sarged; pred=%v, vector_pred=%v, sarg_keys=%v, error=%v",
					pred, vpred, se.sargKeys, err)
			} else {
				errMsg = fmt.Sprintf("Sargable index not sarged; pred=%v, sarg_keys=%v, error=%v",
					pred, se.sargKeys, err)
			}
			return errors.NewPlanError(nil, errMsg)
		}

		se.spans = spans
		if exactSpans && !useIndex2API(se.index, this.context.IndexApiVersion()) {
			exactSpans = spans.ExactSpan1(len(se.keys))
		}
		se.exactSpans = exactSpans
		se.includeSpans = includeSpans
		se.exactIncludes = exactIncludes

		if isOrPred && useFilters {
			se.SetFlags(IE_OR_USE_FILTERS, true)
		}
	}

	return nil
}

func indexHasArrayIndexKey(index datastore.Index) bool {
	return hasArrayIndexKey(index.RangeKey())
}

func hasArrayIndexKey(keys expression.Expressions) bool {
	for _, sk := range keys {
		if isArray, _, _ := sk.IsArrayIndexKey(); isArray {
			return true
		}
	}
	return false
}

func hasRedundantSargKeys(keys datastore.IndexKeys, skeys []bool) bool {
	for i := 0; i < len(keys); i++ {
		if i >= len(skeys) || !skeys[i] {
			continue
		}
		for j := i + 1; j < len(keys); j++ {
			if j >= len(skeys) || !skeys[j] {
				continue
			}
			if keys[i].Expr.DependsOn(keys[j].Expr) || keys[j].Expr.DependsOn(keys[i].Expr) {
				return true
			}
		}
	}
	return false
}

func (this *builder) chooseIntersectScan(sargables map[datastore.Index]*indexEntry,
	node *algebra.KeyspaceTerm, vector bool) map[datastore.Index]*indexEntry {

	keyspace, err := this.getTermKeyspace(node)
	if err != nil {
		if vector {
			// choose random one
			for i, e := range sargables {
				return map[datastore.Index]*indexEntry{
					i: e,
				}
			}
		}
		return sargables
	}

	nTerms := 0
	if this.order != nil {
		nTerms = len(this.order.Terms())
	}

	baseKeyspace, _ := this.baseKeyspaces[node.Alias()]
	return optChooseIntersectScan(keyspace, sargables, nTerms, baseKeyspace, this.limit, this.offset,
		this.advisorValidate(), len(this.baseKeyspaces) == 1, vector, this.context)
}

func bestIndexBySargableKeys(se, te *indexEntry, snc, tnc int, shortest bool) *indexEntry {
	be := bestIndexBySargKeys(se, te, snc, tnc)
	if !shortest && be != nil {
		s_pushdown := se.PushDownProperty()
		t_pushdown := te.PushDownProperty()
		s_redundantSargKey := se.HasFlag(IE_HAS_REDUNDANT_SARG_KEY)
		t_redundantSargKey := te.HasFlag(IE_HAS_REDUNDANT_SARG_KEY)
		if be == se && s_redundantSargKey {
			if t_pushdown > s_pushdown {
				be = te
			} else if t_pushdown == s_pushdown && !t_redundantSargKey {
				be = nil
			}
		} else if be == te && t_redundantSargKey {
			if s_pushdown > t_pushdown {
				be = se
			} else if s_pushdown == t_pushdown && !s_redundantSargKey {
				be = nil
			}
		}
	}
	return be
}

func bestIndexBySargKeys(se, te *indexEntry, snc, tnc int) *indexEntry {
	si := 0
	ti := 0
	for si < len(se.skeys) && ti < len(te.skeys) {
		// for include columns, since the ordering of include columns is not important, if we
		// find an entry corresponding to a "non-sargable" include column, simply skip it
		if si >= len(se.idxKeys) && !se.skeys[si] {
			si++
			continue
		}
		if ti >= len(te.idxKeys) && !te.skeys[ti] {
			ti++
			continue
		}
		if se.skeys[si] != te.skeys[ti] {
			if se.skeys[si] {
				if tnc == 0 {
					return se
				}
				tnc--
				si++
			} else if te.skeys[ti] {
				if snc == 0 {
					return te
				}
				snc--
				ti++
			}
		} else {
			si++
			ti++
		}
	}

	for ; si < len(se.skeys); si++ {
		if se.skeys[si] {
			if tnc == 0 {
				return se
			}
			tnc--
		}
	}

	for ; ti < len(te.skeys); ti++ {
		if te.skeys[ti] {
			if snc == 0 {
				return te
			}
			snc--
		}
	}

	if tnc > snc {
		return te
	} else if snc > tnc {
		return se
	}

	return nil
}

func overlapSpans(expr expression.Expression) bool {
	return expr != nil && expr.MayOverlapSpans()
}

func (this *builder) eqJoinFilter(fl *base.Filter, alias string) (bool, expression.Expression, expression.Expression,
	*base.BaseKeyspace) {

	fltrExpr := fl.FltrExpr()
	eqFltr, ok := fltrExpr.(*expression.Eq)
	if !ok {
		return false, nil, nil, nil
	}

	keyspaces := fl.OrigKeyspaces()
	if len(keyspaces) != 2 {
		return false, nil, nil, nil
	}

	first := eqFltr.First()
	second := eqFltr.Second()
	firstRefs, err1 := expression.CountKeySpaces(first, keyspaces)
	secondRefs, err2 := expression.CountKeySpaces(second, keyspaces)
	if err1 != nil || err2 != nil {
		return false, nil, nil, nil
	}
	if len(firstRefs) != 1 || len(secondRefs) != 1 {
		return false, nil, nil, nil
	}

	var joinAlias string
	for a, _ := range keyspaces {
		if a != alias {
			joinAlias = a
			break
		}
	}

	joinKeyspace, ok := this.baseKeyspaces[joinAlias]
	if !ok || joinKeyspace.IsOuter() {
		return false, nil, nil, nil
	}

	if _, ok = firstRefs[alias]; ok {
		return true, first, second, joinKeyspace
	} else if _, ok = secondRefs[alias]; ok {
		return true, second, first, joinKeyspace
	}
	return false, nil, nil, nil
}

func (this *builder) getIndexFilters(entry *indexEntry, node *algebra.KeyspaceTerm,
	baseKeyspace *base.BaseKeyspace, id expression.Expression) (err error) {

	// special case, if the span is an empty span, no need to proceed
	if isSpecialSargSpan(entry.spans, plan.RANGE_EMPTY_SPAN) {
		return nil
	}

	alias := node.Alias()
	useCBO := this.useCBO && this.keyspaceUseCBO(alias)
	advisorValidate := this.advisorValidate()
	requestId := this.context.RequestId()
	if useCBO {
		// limit_cost indicates whether cost needs to be recalculated due to LIMIT pushdown
		limit_cost := entry.IsPushDownProperty(_PUSHDOWN_LIMIT|_PUSHDOWN_OFFSET) &&
			!entry.HasFlag(IE_LIMIT_OFFSET_COST) && this.limit != nil
		if (entry.cost <= 0.0 || entry.cardinality <= 0.0 || entry.size <= 0 || entry.frCost <= 0.0) || limit_cost {
			var limit, offset int64
			if limit_cost {
				limit, offset = this.getLimitOffset(entry, this.limit, this.offset)
			}
			cost, selec, card, size, frCost, e := indexScanCost(entry, entry.sargKeys, entry.sargIncludes,
				requestId, entry.spans, entry.includeSpans, alias, baseKeyspace.Keyspace(),
				limit, offset, advisorValidate, this.context)
			if e != nil || (cost <= 0.0 || card <= 0.0 || size <= 0 || frCost <= 0.0) {
				useCBO = false
			} else {
				entry.cardinality, entry.selectivity, entry.cost, entry.frCost, entry.size = card, selec, cost, frCost, size
				fetchCost, _, _ := getFetchCost(baseKeyspace.Keyspace(), card)
				if fetchCost > 0.0 {
					entry.fetchCost = fetchCost
				} else {
					useCBO = false
				}
			}
			if limit_cost {
				entry.SetFlags(IE_LIMIT_OFFSET_COST, true)
			}
		}
	}

	includeJoin := true
	if this.joinEnum() {
		if !this.hasBuilderFlag(BUILDER_DO_JOIN_FILTER) {
			includeJoin = false
		}
	} else if baseKeyspace.IsOuter() || baseKeyspace.IsUnnest() || baseKeyspace.HasNoJoinFilterHint() {
		includeJoin = false
	} else if node.IsAnsiJoinOp() && !this.hasBuilderFlag(BUILDER_UNDER_HASH) {
		includeJoin = false
	} else if !useCBO && !baseKeyspace.HasJoinFilterHint() {
		includeJoin = false
	}

	selec := OPT_SELEC_NOT_AVAIL
	if useCBO {
		selec = 1.0
	}

	index := entry.index

	var indexFilters expression.Expressions
	var hasIndexJoinFilters bool
	filters := baseKeyspace.Filters()
	joinFilters := baseKeyspace.JoinFilters()
	nFilters := len(filters)
	nJoinFilters := len(joinFilters)
	pred := baseKeyspace.DnfPred()
	isOrPred := false
	if _, ok := pred.(*expression.Or); ok {
		isOrPred = true
	}

	for _, fl := range filters {
		if fl.IsJoin() {
			nFilters--
			nJoinFilters++
		}
	}
	if !includeJoin {
		nJoinFilters = 0
	}

	if (nFilters + nJoinFilters) == 0 {
		return
	}

	if nFilters > 0 {
		indexFilters = make(expression.Expressions, 0, nFilters)
	}

	// skip array index keys
	arrayKey := entry.HasFlag(IE_ARRAYINDEXKEY)
	idxKeys := entry.idxKeys
	includes := entry.includes
	if entry.HasFlag(IE_VECTOR_KEY_SARGABLE) && !entry.IsPushDownProperty(_PUSHDOWN_ORDER) {
		idxKeys, _, err = replaceVectorKey(idxKeys, entry, false)
		if err != nil {
			return
		}
	}
	coverExprs := make(expression.Expressions, 0, len(idxKeys)+len(includes)+1)
	for _, key := range idxKeys {
		if isArray, _, _ := key.Expr.IsArrayIndexKey(); !isArray && !key.HasAttribute(datastore.IK_VECTOR) {
			if ann, ok := key.Expr.(*expression.ApproxVectorDistance); ok && ann.ReRank() != nil {
				rerankVal := ann.ReRank().Value()
				// assume we need to rerank if value is not known
				if rerankVal == nil || (rerankVal.Type() == value.BOOLEAN && rerankVal.Truth()) {
					// if rerank requested, do not use early order
					continue
				}
			}
			coverExprs = append(coverExprs, key.Expr)
		}
	}
	if len(includes) > 0 {
		coverExprs = append(coverExprs, includes...)
	}
	if !index.IsPrimary() && id != nil {
		coverExprs = append(coverExprs, id)
	}

	if entry.cond != nil {
		fc := make(map[expression.Expression]value.Value, 2)
		fc = entry.cond.FilterExpressionCovers(fc)
		fc = entry.origCond.FilterExpressionCovers(fc)
		filterCovers := mapFilterCovers(fc, false)
		for c, _ := range filterCovers {
			coverExprs = append(coverExprs, c.Covered())
		}
	}

	namedArgs := this.context.NamedArgs()
	positionalArgs := this.context.PositionalArgs()

	if util.IsFeatureEnabled(this.context.FeatureControls(), util.N1QL_EARLY_ORDER) && !arrayKey &&
		this.order != nil && this.limit != nil &&
		!this.hasBuilderFlag(BUILDER_ORDER_DEPENDS_ON_LET) &&
		!entry.IsPushDownProperty(_PUSHDOWN_ORDER|_PUSHDOWN_LIMIT|_PUSHDOWN_OFFSET) &&
		!entry.HasFlag(IE_VECTOR_RERANK) &&
		(len(this.baseKeyspaces) == 1) {

		cons := true
		isParam := false
		nlimit, noffset := this.getLimitOffset(entry, this.limit, this.offset)
		if nlimit <= 0 {
			cons = false
			switch this.limit.(type) {
			case *algebra.NamedParameter, *algebra.PositionalParameter:
				isParam = true
			}
		} else if this.offset != nil && noffset <= 0 {
			cons = false
		}
		if (cons && nlimit > 0) || isParam {
			doOrder := true
			sortTerms := this.order.Expressions()
			sortExprs := make(expression.Expressions, 0, len(sortTerms))
			newTerms, found, err := algebra.ReplaceProjectionAlias(sortTerms, this.projection)
			if err != nil {
				return err
			}

			if !found || this.projection != nil {
				if found {
					sortTerms = newTerms
				}
				for _, sortExpr := range sortTerms {
					if expression.IsCovered(sortExpr, alias, coverExprs, false) {
						sortExprs = append(sortExprs, sortExpr)
					} else {
						doOrder = false
						break
					}
				}
			} else {
				doOrder = false
			}

			if doOrder {
				entry.orderExprs = sortExprs
				entry.SetFlags(IE_HAS_EARLY_ORDER, true)
				if useCBO {
					var sortCard float64
					if entry.IsPushDownProperty(_PUSHDOWN_EXACTSPANS) {
						if cons {
							sortCard = float64(nlimit + noffset)
						} else {
							// in case limit/offset is not constant, for
							// costing purpose here assume half of the
							// documents from the index scan are used for
							// fetch
							sortCard = 0.5 * entry.cardinality
						}
						if sortCard > entry.cardinality {
							sortCard = entry.cardinality
						}
					} else {
						sortCard = entry.cardinality
					}
					fetchCost, _, _ := getFetchCost(baseKeyspace.Keyspace(), sortCard)
					if fetchCost > 0.0 {
						entry.fetchCost = fetchCost
					} else {
						useCBO = false
					}
				}
			}
		}
	}

	if !isPushDownProperty(entry.pushDownProperty, _PUSHDOWN_EXACTSPANS) || len(includes) > 0 {
		missing := entry.HasFlag(IE_LEADINGMISSING)
		skip := useSkipIndexKeys(index, this.context.IndexApiVersion())
		chkOr := isOrPred && !entry.HasFlag(IE_OR_USE_FILTERS)
		chkUnnest := entry.HasFlag(IE_ARRAYINDEXKEY_SARGABLE) && len(entry.unnestAliases) > 0
	filterloop:
		for _, fl := range filters {
			if fl.HasSubq() {
				continue
			} else if fl.IsUnnest() {
				if !chkUnnest {
					continue
				}
				// in case there are multiple unnests, make sure the filter is for
				// the unnest alias that this unnest scan is for
				for ks, _ := range fl.Keyspaces() {
					if ks == alias {
						// allow reference to primary keyspace
						continue
					}
					found := false
					for _, un := range entry.unnestAliases {
						if ks == un {
							found = true
							break
						}
					}
					if !found {
						continue filterloop
					}
				}
			}
			fltrExpr := fl.FltrExpr()
			derived := false
			orig := false
			subFltr := false
			if chkOr || chkUnnest {
				fltr := this.orGetIndexFilter(fltrExpr, entry.index, entry.sargKeys, baseKeyspace, missing, skip)
				if fltr == nil {
					continue
				} else if fltr != fltrExpr {
					fltrExpr = fltr
					subFltr = true
				}
			} else if _, ok := entry.exactFilters[fl]; ok {
				// Skip the filters used to generate exact spans since these are
				// supposedly evaluated by the indexer already.
				// It is possible that a span starts out exact but was turned to non-exact
				// later, but this will not cause wrong result (just potential inefficiency)
				continue
			}

			if !subFltr && entry.cond != nil {
				// Also skip filters that is in index condition
				origExpr := fl.OrigExpr()
				flExpr := fltrExpr
				if len(namedArgs) > 0 || len(positionalArgs) > 0 {
					flExpr, err = base.ReplaceParameters(flExpr, namedArgs, positionalArgs)
					if err != nil {
						return
					}
					if origExpr != nil {
						origExpr, err = base.ReplaceParameters(origExpr, namedArgs, positionalArgs)
						if err != nil {
							return
						}
					}
				}
				if base.SubsetOf(entry.cond, flExpr) {
					continue
				}
				if origExpr != nil && base.SubsetOf(entry.origCond, origExpr) {
					continue
				}
			}
			if base.IsDerivedExpr(fltrExpr) && !fl.IsJoin() {
				derived = true
				if fl.OrigExpr() != nil {
					fltrExpr = fl.OrigExpr()
					orig = true
				}
			}
			covered := expression.IsCovered(fltrExpr, alias, coverExprs, false)
			if covered && chkUnnest && fl.IsUnnest() {
				// for unnest scan, the array index key is replaced with the unnested
				// array key representation (see getUnnestSargKeys()), thus is already
				// present in coverExprs
				for _, unAlias := range entry.unnestAliases {
					covered = expression.IsCovered(fltrExpr, unAlias, coverExprs, false)
					if !covered {
						break
					}
				}
			}
			if covered {
				if !fl.IsJoin() {
					if useCBO {
						if fl.Selec() > 0.0 {
							// filters marked with EXPR_DEFAULT_LIKE has its
							// selectivity already accounted for in spans
							if !subFltr && !fltrExpr.HasExprFlag(expression.EXPR_DEFAULT_LIKE) {
								if _, ok := entry.exactIncludeFilters[fl]; !ok {
									selec *= fl.Selec()
								}
							}
						} else {
							useCBO = false
							selec = OPT_SELEC_NOT_AVAIL
						}
					}
					if !derived || orig {
						indexFilters = append(indexFilters, fltrExpr)
					}
				} else if includeJoin {
					eq, self, other, joinKeyspace := this.eqJoinFilter(fl, alias)
					if eq && self != nil && other != nil && joinKeyspace != nil {
						hasIndexJoinFilters = true
						baseKeyspace.AddBFSource(joinKeyspace, index, self, other, fl)
					}
				}
			}
		}
	}

	if len(indexFilters) > 0 {
		entry.indexFilters = indexFilters
		entry.SetFlags(IE_HAS_FILTER, true)
		if useCBO {
			entry.cost, entry.cardinality, entry.size, entry.frCost = getSimpleFilterCost(alias,
				entry.cost, entry.cardinality, selec, entry.size, entry.frCost)
			entry.selectivity *= selec
			fetchCost, _, _ := getFetchCost(baseKeyspace.Keyspace(), entry.cardinality)
			if fetchCost > 0.0 {
				entry.fetchCost = fetchCost
			} else {
				useCBO = false
			}
		}
	}

	if !includeJoin {
		return
	}

	for _, fl := range joinFilters {
		fltrExpr := fl.FltrExpr()
		if expression.IsCovered(fltrExpr, alias, coverExprs, false) {
			eq, self, other, joinKeyspace := this.eqJoinFilter(fl, alias)
			if eq && self != nil && other != nil && joinKeyspace != nil {
				hasIndexJoinFilters = true
				baseKeyspace.AddBFSource(joinKeyspace, index, self, other, fl)
			}
		}
	}

	if hasIndexJoinFilters {
		entry.SetFlags(IE_HAS_JOIN_FILTER, true)
	}

	return
}

func (this *builder) buildIndexFilters(entry *indexEntry, baseKeyspace *base.BaseKeyspace,
	id expression.Expression, node *algebra.KeyspaceTerm) (
	covers expression.Covers, filterCovers map[*expression.Cover]value.Value,
	filter expression.Expression, indexProjection *plan.IndexProjection, err error) {

	useCBO := this.useCBO && this.keyspaceUseCBO(node.Alias())
	if useCBO && (entry.cost <= 0.0 || entry.cardinality <= 0.0 || entry.size <= 0 || entry.frCost <= 0.0) {
		useCBO = false
	}

	var indexFilters, joinFilters, sortExprs expression.Expressions
	var bfCost, bfFrCost, bfSelec float64
	hasVector := entry.HasFlag(IE_VECTOR_KEY_SARGABLE)
	if entry.HasFlag(IE_HAS_FILTER) {
		indexFilters = entry.indexFilters
	}
	if !hasVector && entry.HasFlag(IE_HAS_JOIN_FILTER) {
		if useCBO {
			bfSelec, bfCost, bfFrCost, joinFilters = optChooseJoinFilters(baseKeyspace, entry.index)
			if len(joinFilters) > 0 && (bfSelec <= 0.0 || bfCost <= 0.0 || bfFrCost <= 0.0) {
				useCBO = false
			}
		} else {
			joinFilters = baseKeyspace.GetAllJoinFilterExprs(entry.index)
		}
	}
	if entry.HasFlag(IE_HAS_EARLY_ORDER) {
		sortExprs = entry.orderExprs
	}

	keys := entry.idxKeys
	includes := entry.includes
	if hasVector && entry.HasFlag(IE_HAS_EARLY_ORDER) {
		keys, _, err = replaceVectorKey(keys, entry, false)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}
	if len(indexFilters) > 0 || len(joinFilters) > 0 || len(sortExprs) > 0 {
		allFilters := indexFilters
		if len(joinFilters) > 0 {
			allFilters = append(allFilters, joinFilters...)
		}
		if len(sortExprs) > 0 {
			allFilters = append(allFilters, sortExprs...)
		}
		indexProjection = this.buildIndexProjection(entry, allFilters, id, true, nil)
		if indexProjection != nil {
			covers = make(expression.Covers, 0, len(indexProjection.EntryKeys)+1)
			for _, i := range indexProjection.EntryKeys {
				if i < len(keys) {
					if !keys[i].HasAttribute(datastore.IK_VECTOR) {
						covers = append(covers, expression.NewIndexKey(keys[i].Expr))
					}
				} else if i < len(keys)+len(includes) {
					covers = append(covers, expression.NewIndexKey(includes[i-len(keys)]))
				} else {
					return nil, nil, nil, nil, errors.NewPlanInternalError(fmt.Sprintf("buildIndexFilters: index projection "+
						"key position %d beyond key length(%d)", i, len(keys)))
				}
			}
			covers = append(covers, expression.NewIndexKey(id))
		} else {
			covers = make(expression.Covers, 0, len(keys)+len(includes))
			for _, key := range keys {
				if !key.HasAttribute(datastore.IK_VECTOR) {
					covers = append(covers, expression.NewIndexKey(key.Expr))
				}
			}
			for _, include := range includes {
				covers = append(covers, expression.NewIndexKey(include))
			}
			if !entry.index.IsPrimary() {
				covers = append(covers, expression.NewIndexKey(id))
			}
		}

		if entry.cond != nil {
			fc := make(map[expression.Expression]value.Value, 2)
			fc = entry.cond.FilterExpressionCovers(fc)
			fc = entry.origCond.FilterExpressionCovers(fc)
			filterCovers = mapFilterCovers(fc, false)
		}
	}

	if len(indexFilters) > 0 {
		if len(indexFilters) == 1 {
			filter = indexFilters[0].Copy()
		} else {
			filter = expression.NewAnd(indexFilters.Copy()...)
		}
		// OK to always do cover transformation since 'filter' should be a copy
		if len(covers) > 0 || len(filterCovers) > 0 {
			// no array index keys so can Map directly
			coverer := expression.NewCoverer(covers, filterCovers)
			filter, err = coverer.Map(filter)
			if err != nil {
				return nil, nil, nil, nil, err
			}
		}

		if useCBO {
			cost, cardinality, size, frCost := getIndexProjectionCost(entry.index, indexProjection, entry.cardinality)

			if cost > 0.0 && cardinality > 0.0 && size > 0 && frCost > 0.0 {
				entry.cost += cost
				entry.cardinality = cardinality
				entry.size += size
				entry.frCost += frCost
			} else {
				useCBO = false
				entry.cost, entry.cardinality, entry.frCost, entry.size = OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL,
					OPT_COST_NOT_AVAIL, OPT_SIZE_NOT_AVAIL
			}
		}
	}

	if useCBO && len(joinFilters) > 0 {
		entry.cost += bfCost
		entry.frCost += bfFrCost
		entry.cardinality *= bfSelec
		fetchCost, _, _ := getFetchCost(baseKeyspace.Keyspace(), entry.cardinality)
		if fetchCost > 0.0 {
			entry.fetchCost = fetchCost
		}
	}

	return
}

func getIndexAllIndexes(indexes map[datastore.Index]*indexEntry, baseKeyspace *base.BaseKeyspace) map[datastore.Index]*indexEntry {
	var indexNames []string
	for _, hint := range baseKeyspace.IndexHints() {
		if idxAll, ok := hint.(*algebra.HintIndexAll); ok {
			indexNames = idxAll.Indexes()
		}
	}

	for index, _ := range indexes {
		found := false
		for _, indexName := range indexNames {
			if index.Name() == indexName {
				found = true
				break
			}
		}
		if !found {
			delete(indexes, index)
		}
	}

	return indexes
}

func (this *builder) orSargUseFilters(pred *expression.Or, vpred expression.Expression,
	baseKeyspace *base.BaseKeyspace, entry *indexEntry) bool {

	skip := useSkipIndexKeys(entry.index, this.context.IndexApiVersion())
	missing := entry.HasFlag(IE_LEADINGMISSING)

	// if all subterms of OR gives the same set of sargable keys, use individual filters
	var min, max int
	var skeys []bool
	for i, child := range pred.Operands() {
		cmin, cmax, _, _, cskeys := SargableFor(child, vpred, entry.index, entry.idxSargKeys,
			nil, missing, skip, nil, this.context, this.aliases)
		if i == 0 {
			min = cmin
			max = cmax
			skeys = cskeys
		} else {
			if cmin != min || cmax != max || len(cskeys) != len(skeys) {
				return false
			}
			for j := 0; j < len(skeys); j++ {
				if cskeys[j] != skeys[j] {
					return false
				}
			}
		}
	}

	nSargKeys := 0
	for j := 0; j < len(skeys); j++ {
		if skeys[j] {
			nSargKeys++
		}
	}
	if nSargKeys > 1 {
		// if each OR subterm contains more than one sargable key, further check the
		// individual filters to see
		//    1. whether there are multiple OR-filters,
		//    2. for each individual OR-filter, whether it can sarg multiple keys.
		// if either is true, then we cannot use individual filters to sarg, otherwise
		// we may end up with multiple spans for each index key, and combining
		// multiple such spans will result in cross-muptiplication of the spans.
		numOr := 0
		for _, filter := range baseKeyspace.Filters() {
			fltrExpr := filter.FltrExpr()
			or, ok := fltrExpr.(*expression.Or)
			if !ok {
				continue
			}
			numOr++
			if numOr > 1 {
				return false
			}
			nsarg := 0
			for i, key := range entry.idxSargKeys {
				cmissing := missing
				if i > 0 {
					cmissing = true
				}
				cmin, _, _, _, _ := SargableFor(or, nil, entry.index, datastore.IndexKeys{key},
					nil, cmissing, skip, nil, this.context, this.aliases)
				if cmin > 0 {
					nsarg++
					if nsarg > 1 {
						// this is attepmting to diffrentiate between:
						//   c1 = 1 AND (c2 = 2 OR c2 = 3)
						// vs
						//   (c1 = 1 AND c2 = 2) OR (c1 = 2 AND c2 = 3)
						// we can allow using of individual filters for
						// sarging of indexes in the first case but not
						// in the second case
						// this does leave
						//   (c1 = 1 AND c2 = 2) OR (c1 = 1 AND c2 = 3)
						// which is equivalent to the first one also not
						// able to use individual filters tos sarg, since
						// we don't detect actual duplicated filter in
						// subarms of OR currently.
						return false
					}
				}
			}
		}
	}

	return true
}

func (this *builder) orGetIndexFilter(pred expression.Expression, index datastore.Index, keys expression.Expressions,
	baseKeyspace *base.BaseKeyspace, missing, skip bool) expression.Expression {
	var orOps expression.Expressions
	if or, ok := pred.(*expression.Or); ok {
		orOps = or.Operands()
	} else {
		orOps = expression.Expressions{pred}
	}

	modified := false
	terms := make(expression.Expressions, 0, len(orOps))
	for _, op := range orOps {
		var term expression.Expression
		var andOps expression.Expressions
		if and, ok := op.(*expression.And); ok {
			andOps = and.Operands()
		} else {
			andOps = expression.Expressions{op}
		}
		for _, op1 := range andOps {
			add := true
			for i, key := range keys {
				min, _, _, _, _ := SargableFor(op1, nil, index, datastore.IndexKeys{&datastore.IndexKey{key, datastore.IK_NONE}},
					nil, (missing || i > 0), skip, nil, this.context, this.aliases)
				if min == 0 {
					continue
				}
				rs, exact, err := sargFor(op1, index, key, false, false, baseKeyspace,
					this.keyspaceNames, this.advisorValidate(), (missing || i > 0),
					false, false, false, i, this.aliases, this.context)
				if err == nil && rs != nil && exact {
					add = false
					break
				}
			}
			if add {
				if term == nil {
					term = op1
				} else {
					term = expression.NewAnd(term, op1)
				}
			} else {
				modified = true
			}
		}
		if term == nil {
			// one of the OR subterms is "empty", i.e., true
			return nil
		}
		found := false
		if modified {
			// avoid adding redundant term
			for i := 0; i < len(terms); i++ {
				// SubsetOf() could result in exponential number of comparisons if
				// both term and terms[i] are AND expressions with many children
				if _, ok := term.(*expression.And); ok {
					if term.EquivalentTo(terms[i]) {
						found = true
					}
				} else {
					if base.SubsetOf(terms[i], term) {
						terms[i] = term
						found = true
					}
				}
				if found {
					break
				}
			}
		}
		if !found {
			terms = append(terms, term)
		}
	}

	if !modified {
		return pred
	}

	var rv expression.Expression
	if len(terms) == 0 {
		return nil
	} else if len(terms) == 1 {
		rv = terms[0]
	} else {
		rv = expression.NewOr(terms...)
	}

	return rv
}

func (this *builder) getLimitOffset(entry *indexEntry, limit, offset expression.Expression) (int64, int64) {
	namedArgs := this.context.NamedArgs()
	positionalArgs := this.context.PositionalArgs()

	nlimit := int64(-1)
	noffset := int64(-1)
	if len(namedArgs) > 0 || len(positionalArgs) > 0 {
		var err error
		limit, err = base.ReplaceParameters(limit, namedArgs, positionalArgs)
		if err != nil {
			return -1, -1
		}
		if offset != nil {
			offset, err = base.ReplaceParameters(offset, namedArgs, positionalArgs)
			if err != nil {
				return -1, -1
			}
		}
	}

	if entry.IsPushDownProperty(_PUSHDOWN_LIMIT) && entry.HasFlag(IE_VECTOR_RERANK) {
		factor := 0
		if index6, ok := entry.index.(datastore.Index6); ok {
			factor = int(index6.RerankFactor())
		}
		if factor <= 0 {
			factor = plan.RERANK_FACTOR
		}
		limit = expandOffsetLimit(offset, limit, factor)
	}

	lv, static := base.GetStaticInt(limit)
	if static {
		nlimit = lv
	}
	if offset != nil {
		ov, static := base.GetStaticInt(offset)
		if static {
			noffset = ov
		}
	}
	return nlimit, noffset
}
