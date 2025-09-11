//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

// Covering Scan

func (this *builder) buildCovering(indexes, unnestIndexes, flex map[datastore.Index]*indexEntry,
	node *algebra.KeyspaceTerm, baseKeyspace *base.BaseKeyspace, subset, id expression.Expression,
	searchSargables []*indexEntry, unnests []*algebra.Unnest) (
	scan plan.SecondaryScan, sargLength int, err error) {

	// covering turrned off or ANSI NEST, or system keyspace
	if this.cover == nil || node.IsAnsiNest() || baseKeyspace.IsSystem() {
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
	// GSI Unnest covering scan
	if len(unnests) > 0 && len(unnestIndexes) > 0 {
		scan, sargLength, err = this.buildCoveringUnnestScan(node, baseKeyspace.DnfPred(),
			baseKeyspace.OrigPred(), subset, id, unnestIndexes, unnests)
		if scan != nil || err != nil {
			return
		}
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
	exprs := this.getExprsToCover()
	pred := baseKeyspace.DnfPred()
	origPred := baseKeyspace.OrigPred()
	useCBO := this.useCBO && this.keyspaceUseCBO(alias)

	narrays := 0
	coveringEntries := _COVERING_ENTRY_POOL.Get()
	defer _COVERING_ENTRY_POOL.Put(coveringEntries)

outer:
	for index, entry := range indexes {
		if !useCBO && entry.arrayKey != nil && narrays < len(coveringEntries) {
			continue
		}

		idxKeys := entry.idxKeys
		keys := entry.keys
		vector := entry.HasFlag(IE_VECTOR_KEY_SARGABLE)
		var vecExpr *expression.ApproxVectorDistance
		var err error

		// Matches execution.spanScan.RunOnce()
		if !index.IsPrimary() {
			idxKeys = append(idxKeys, &datastore.IndexKey{id, datastore.IK_NONE})
			keys = append(keys, id)
		}

		if vector {
			idxKeys, vecExpr, err = replaceVectorKey(idxKeys, entry, true)
			if err != nil {
				return nil, 0, err
			}
		}

		// Include filter covers
		coveringExprs, filterCovers, err := indexCoverExpressions(entry, idxKeys, true, pred,
			origPred, alias, this.context)
		if err != nil {
			return nil, 0, err
		}

		implicitAny := implicitAnyCover(entry, true, this.context.FeatureControls())

		// Skip non-covering index
		for _, expr := range exprs {
			if !expression.IsCovered(expr, alias, coveringExprs, implicitAny) {
				continue outer
			}
		}

		var implcitIndexProj map[int]bool
		if implicitAny {
			mapAnys, err1 := expression.GatherAny(exprs, entry.arrayKey, false)
			if err1 != nil {
				continue
			}
			ifc := implicitFilterCovers(entry.arrayKey)
			if len(ifc) > 0 {
				if len(filterCovers) == 0 {
					filterCovers = ifc
				} else {
					for c, v := range ifc {
						if _, ok := filterCovers[c]; !ok {
							filterCovers[c] = v
						}
					}
				}
			}

			idxKeys = replaceFlattenKeys(idxKeys, entry)
			implcitIndexProj = implicitIndexKeysProj(idxKeys, mapAnys)
		}

		if entry.arrayKey != nil {
			narrays++
		}

		entry.pushDownProperty = this.indexPushDownProperty(entry, keys, nil, pred, origPred,
			alias, nil, false, true, (len(this.baseKeyspaces) == 1), implicitAny)

		// in vector query, if rerank is requested but ORDER/LIMIT cannot be pushed down,
		// then rerank cannot be done in the index, need to Fetch in this case (to rerank)
		if vector && vecExpr != nil && vecExpr.HasReRank(true) &&
			(!entry.IsPushDownProperty(_PUSHDOWN_ORDER) || !entry.IsPushDownProperty(_PUSHDOWN_LIMIT)) {
			continue outer
		}

		coveringEntries[index] = &coveringEntry{
			idxEntry:         entry,
			filterCovers:     filterCovers,
			implcitIndexProj: implcitIndexProj,
			implicitAny:      implicitAny,
			indexKeys:        idxKeys,
		}
	}

	// No covering index available
	if len(coveringEntries) == 0 {
		return nil, 0, nil
	}

	index := this.bestCoveringIndex(useCBO, baseKeyspace.Name(), baseKeyspace.Keyspace(),
		coveringEntries, (narrays < len(coveringEntries)))
	coveringEntry := coveringEntries[index]
	keys := coveringEntry.indexKeys
	var implcitIndexProj map[int]bool
	if coveringEntry.implicitAny {
		implcitIndexProj = coveringEntry.implcitIndexProj
	}

	var includes expression.Expressions
	if coveringEntry.idxEntry != nil {
		includes = coveringEntry.idxEntry.includes
	}

	// Include covering expression from index keys
	covers := make(expression.Covers, 0, len(keys)+len(includes))
	size := len(keys)
	for i, key := range keys {
		if i == (size-1) && !index.IsPrimary() {
			for _, include := range includes {
				covers = append(covers, expression.NewCover(include))
			}
			covers = append(covers, expression.NewCover(key.Expr))
		} else {
			covers = append(covers, expression.NewCover(key.Expr))
		}
	}

	return this.buildCreateCoveringScan(coveringEntry.idxEntry, node, id, pred, exprs, keys, false,
		coveringEntry.idxEntry.arrayKey != nil, coveringEntry.implicitAny, covers,
		coveringEntry.filterCovers, implcitIndexProj)
}

func (this *builder) bestCoveringIndex(useCBO bool, alias, keyspace string,
	coveringEntries map[datastore.Index]*coveringEntry, noArray bool) (index datastore.Index) {

	hasGroupAggs := false
	hasOrder := false

	if useCBO {
		for _, ce := range coveringEntries {
			entry := ce.idxEntry
			// limit_cost indicates whether cost needs to be recalculated due to LIMIT pushdown
			limit_cost := entry.IsPushDownProperty(_PUSHDOWN_LIMIT|_PUSHDOWN_OFFSET) &&
				!entry.HasFlag(IE_LIMIT_OFFSET_COST) && this.limit != nil
			if entry.cost <= 0.0 || limit_cost {
				var limit, offset int64
				if limit_cost {
					limit, offset = this.getLimitOffset(entry, this.limit, this.offset)
				}
				cost, selec, card, size, frCost, e := indexScanCost(entry, entry.sargKeys,
					entry.sargIncludes, this.context.RequestId(), entry.spans, entry.includeSpans,
					alias, keyspace, limit, offset, this.advisorValidate(), this.context)
				if e != nil || (cost <= 0.0 || card <= 0.0 || size <= 0 || frCost <= 0.0) {
					useCBO = false
				} else {
					entry.cardinality, entry.cost, entry.frCost, entry.size, entry.selectivity = card, cost, frCost, size, selec
				}
				if limit_cost {
					entry.SetFlags(IE_LIMIT_OFFSET_COST, true)
				}
			}

			if entry.IsPushDownProperty(_PUSHDOWN_FULLGROUPAGGS | _PUSHDOWN_GROUPAGGS) {
				hasGroupAggs = true
			}
			if entry.IsPushDownProperty(_PUSHDOWN_ORDER | _PUSHDOWN_PARTIAL_ORDER) {
				hasOrder = true
			}
		}
	}

	var centry *coveringEntry
	var i_cost, i_cardinality float64
	var i_size int64
	var i_pushdown PushDownProperties
	if useCBO {
		// if group/aggregate pushdown and/or order pushdown available, add the corresponding
		// cost for group and/or order to the index cost for those indexes that do not have
		// the appropriate pushdown before comparison
		doGroupAggs := hasGroupAggs
		doOrder := hasOrder
		if doGroupAggs && this.group == nil {
			doGroupAggs = false
		}
		if doOrder && this.order == nil {
			doOrder = false
		}
		for _, ce := range coveringEntries {
			if centry == nil {
				centry = ce
				i_cost = ce.idxEntry.cost
				i_cardinality = ce.idxEntry.cardinality
				i_size = ce.idxEntry.size
				i_pushdown = ce.idxEntry.PushDownProperty()
				if doGroupAggs && !isPushDownProperty(i_pushdown, _PUSHDOWN_FULLGROUPAGGS) {
					costInitial, _, costIntermediate, _, costFinal, _ :=
						getGroupCosts(this.group, this.aggs, i_cost, i_cardinality,
							i_size, this.keyspaceNames, this.maxParallelism)
					if costInitial <= 0.0 || costIntermediate <= 0.0 || costFinal <= 0.0 {
						doGroupAggs = false
					} else {
						i_cost += costInitial + costIntermediate + costFinal
					}
				}

				if doOrder && !isPushDownProperty(i_pushdown, _PUSHDOWN_ORDER) {
					scost, _, _, _ := getSortCost(i_size, len(this.order.Terms()),
						i_cardinality, -1, -1)
					if scost <= 0.0 {
						doOrder = false
					} else {
						i_cost += scost
					}
				}

				continue
			}
			// consider in order:
			//   - cost
			//   - cardinality
			//   - sumKeys
			//   - minKeys
			//   - pushdown property
			c_cost := ce.idxEntry.cost
			c_cardinality := ce.idxEntry.cardinality
			c_size := ce.idxEntry.size
			c_pushdown := ce.idxEntry.PushDownProperty()

			if hasGroupAggs {
				if doGroupAggs && !isPushDownProperty(c_pushdown, _PUSHDOWN_FULLGROUPAGGS) {
					costInitial, _, costIntermediate, _, costFinal, _ :=
						getGroupCosts(this.group, this.aggs, c_cost, c_cardinality,
							c_size, this.keyspaceNames, this.maxParallelism)
					if costInitial <= 0.0 || costIntermediate <= 0.0 || costFinal <= 0.0 {
						doGroupAggs = false
						// reset i_cost to original index cost
						i_cost = centry.idxEntry.cost
					} else {
						c_cost += costInitial + costIntermediate + costFinal
					}
				}
				if !doGroupAggs {
					// if group cost is not available, use the index cost for
					// comparison but treat full/partial groupaggs/order as the
					// same (i.e. use index cost to determine the better one)
					better, similar := comparePushDownProperties(c_pushdown, i_pushdown)
					if better {
						centry = ce
						i_cost = c_cost
						i_cardinality = c_cardinality
						i_size = c_size
						i_pushdown = c_pushdown
						continue
					} else if !similar {
						continue
					}
				}
			}

			if hasOrder {
				if doOrder && !isPushDownProperty(c_pushdown, _PUSHDOWN_ORDER) {
					scost, _, _, _ := getSortCost(c_size, len(this.order.Terms()),
						c_cardinality, -1, -1)
					if scost <= 0.0 {
						doOrder = false
						// reset i_cost/c_cost to original index cost
						i_cost = centry.idxEntry.cost
						c_cost = ce.idxEntry.cost
					} else {
						c_cost += scost
					}
				}
				if !doOrder {
					// if sort cost is not available, use the index cost for
					// comparison but treat full/partial groupaggs/order as the
					// same (i.e. use index cost to determine the better one)
					better, similar := comparePushDownProperties(c_pushdown, i_pushdown)
					if better {
						centry = ce
						i_cost = c_cost
						i_cardinality = c_cardinality
						i_size = c_size
						i_pushdown = c_pushdown
						continue
					} else if !similar {
						continue
					}
				}
			}

			if c_cost != i_cost {
				t_c_cost := c_cost
				t_i_cost := i_cost
				if c_pushdown != i_pushdown && ((hasGroupAggs && !doGroupAggs) || (hasOrder && !doOrder)) {
					// if comparing two indexes with similar index pushdown
					// properties but group/sort cost is not available, just
					// add 10% additional cost to the existing index cost as
					// an estimate of group/sort cost
					if hasGroupAggs && !doGroupAggs {
						t_c_pushdown := c_pushdown & (_PUSHDOWN_FULLGROUPAGGS | _PUSHDOWN_GROUPAGGS)
						t_i_pushdown := i_pushdown & (_PUSHDOWN_FULLGROUPAGGS | _PUSHDOWN_GROUPAGGS)
						if t_c_pushdown > t_i_pushdown {
							t_i_cost += 0.1 * i_cost
						} else if t_c_pushdown < t_i_pushdown {
							t_c_cost += 0.1 * c_cost
						}
					}
					if hasOrder && !doOrder {
						t_c_pushdown := c_pushdown & (_PUSHDOWN_ORDER | _PUSHDOWN_PARTIAL_ORDER)
						t_i_pushdown := i_pushdown & (_PUSHDOWN_ORDER | _PUSHDOWN_PARTIAL_ORDER)
						if t_c_pushdown > t_i_pushdown {
							t_i_cost += 0.1 * i_cost
						} else if t_c_pushdown < t_i_pushdown {
							t_c_cost += 0.1 * c_cost
						}
					}
				}
				if t_c_cost < t_i_cost {
					centry = ce
					i_cost = c_cost
					i_cardinality = c_cardinality
					i_size = c_size
					i_pushdown = c_pushdown
				}
				continue
			}
			if c_cardinality != i_cardinality {
				if c_cardinality < i_cardinality {
					centry = ce
					i_cost = c_cost
					i_cardinality = c_cardinality
					i_size = c_size
					i_pushdown = c_pushdown
				}
				continue
			}
			c_sumKeys := ce.idxEntry.sumKeys + ce.idxEntry.includeKeys
			i_sumKeys := centry.idxEntry.sumKeys + centry.idxEntry.includeKeys
			if c_sumKeys != i_sumKeys {
				if c_sumKeys > i_sumKeys {
					centry = ce
					i_cost = c_cost
					i_cardinality = c_cardinality
					i_size = c_size
					i_pushdown = c_pushdown
				}
				continue
			}
			c_minKeys := ce.idxEntry.minKeys
			if c_minKeys == len(ce.idxEntry.idxKeys) {
				c_minKeys += ce.idxEntry.includeKeys
			}
			i_minKeys := centry.idxEntry.minKeys
			if i_minKeys == len(centry.idxEntry.idxKeys) {
				i_minKeys += centry.idxEntry.includeKeys
			}
			if c_minKeys > i_minKeys {
				centry = ce
				i_cost = c_cost
				i_cardinality = c_cardinality
				i_size = c_size
				i_pushdown = c_pushdown
			}
			if c_pushdown > i_pushdown {
				centry = ce
				i_cost = c_cost
				i_cardinality = c_cardinality
				i_size = c_size
				i_pushdown = c_pushdown
			}
		}
		return centry.idxEntry.index
	}

	// Avoid array indexes if possible
	if noArray {
		for a, ce := range coveringEntries {
			if ce.idxEntry.arrayKey != nil {
				delete(coveringEntries, a)
			}
		}
	}

couter:
	// keep indexes with highest continous sargable indexes
	for sc, _ := range coveringEntries {
		se := coveringEntries[sc].idxEntry
		for tc, _ := range coveringEntries {
			if sc != tc {
				te := coveringEntries[tc].idxEntry
				if be := bestIndexBySargableKeys(se, te, se.nEqCond, te.nEqCond, false); be != nil {
					if be == te {
						delete(coveringEntries, sc)
						continue couter
					}
					delete(coveringEntries, tc)
				}
			}
		}
	}

	// Keep indexes with max sumKeys
	sumKeys := 0
	for _, ce := range coveringEntries {
		if max := ce.idxEntry.sumKeys + ce.idxEntry.nEqCond + ce.idxEntry.includeKeys; max > sumKeys {
			sumKeys = max
		}
	}

	for c, ce := range coveringEntries {
		if ce.idxEntry.sumKeys+ce.idxEntry.nEqCond+ce.idxEntry.includeKeys < sumKeys {
			delete(coveringEntries, c)
		}
	}

	// Keep indexes with max minKeys
	minKeys := 0
	for _, ce := range coveringEntries {
		cminKeys := ce.idxEntry.minKeys
		if cminKeys == len(ce.idxEntry.idxKeys) {
			cminKeys += ce.idxEntry.includeKeys
		}
		if cminKeys > minKeys {
			minKeys = cminKeys
		}
	}

	for c, ce := range coveringEntries {
		cminKeys := ce.idxEntry.minKeys
		if cminKeys == len(ce.idxEntry.idxKeys) {
			cminKeys += ce.idxEntry.includeKeys
		}
		if cminKeys < minKeys {
			delete(coveringEntries, c)
		}
	}

	// vector index
	vector := false
	needRerank := false
	canRerank := false
	sargableIncludes := 0
	for c, ce := range coveringEntries {
		if c6, ok := c.(datastore.Index6); ok && c6.IsVector() {
			vector = true
			if c6.AllowRerank() {
				canRerank = true
			}
			if ce.idxEntry.HasFlag(IE_VECTOR_RERANK) {
				needRerank = true
			}
		}
		if ce.idxEntry.includeKeys > sargableIncludes {
			sargableIncludes = ce.idxEntry.includeKeys
		}
	}
	if vector {
		if needRerank && canRerank {
			// remove indexes with no reranking capability
			for c, _ := range coveringEntries {
				if c6, ok := c.(datastore.Index6); !ok || !c6.AllowRerank() {
					delete(coveringEntries, c)
				}
			}
		}
		if sargableIncludes > 0 {
			// prefer sargable include keys
			for c, ce := range coveringEntries {
				if ce.idxEntry.includeKeys < sargableIncludes {
					delete(coveringEntries, c)
				}
			}
		}
	}

	// Use shortest remaining index
	minLen := 0
	for _, ce := range coveringEntries {
		cLen := len(ce.idxEntry.keys) + len(ce.idxEntry.includes)
		if centry == nil {
			centry = ce
			minLen = cLen
		} else {
			c_pushdown := ce.idxEntry.PushDownProperty()
			i_pushdown := centry.idxEntry.PushDownProperty()
			if (c_pushdown > i_pushdown) ||
				((c_pushdown == i_pushdown) &&
					(cLen < minLen || (cLen == minLen && ce.idxEntry.index.Condition() != nil))) {
				centry = ce
				minLen = cLen
			}
		}
	}
	return centry.idxEntry.index
}

func (this *builder) buildCreateCoveringScan(entry *indexEntry, node *algebra.KeyspaceTerm,
	id, pred expression.Expression, exprs expression.Expressions, keys datastore.IndexKeys,
	unnestScan, arrayIndex, implicitAny bool,
	covers expression.Covers, filterCovers map[*expression.Cover]value.Value,
	idxProj map[int]bool) (plan.SecondaryScan, int, error) {

	sargLength := len(entry.sargKeys)
	useCBO := this.useCBO && this.keyspaceUseCBO(node.Alias())
	baseKeyspace, _ := this.baseKeyspaces[node.Alias()]
	hasDeltaKeyspace := this.context.HasDeltaKeyspace(baseKeyspace.Keyspace())
	countPush := arrayIndex
	array := arrayIndex
	if !unnestScan {
		countPush = !arrayIndex
		array = false
	}

	index := entry.index
	duplicates := entry.spans.CanHaveDuplicates(index, this.context.IndexApiVersion(), overlapSpans(pred), arrayIndex)
	indexProjection := this.buildIndexProjection(entry, exprs, id, index.IsPrimary() || arrayIndex || duplicates, idxProj)

	// Check and reset pagination pushdows
	indexKeyOrders := this.checkResetPaginations(entry, keys, id)

	// Build old Aggregates on Index2 only
	scan := this.buildCoveringPushdDownIndexScan2(entry, node, baseKeyspace, pred, indexProjection,
		countPush, array, covers, filterCovers)
	if scan != nil {
		return scan, sargLength, nil
	}

	// Aggregates check and reset
	var indexGroupAggs *plan.IndexGroupAggregates
	if !entry.IsPushDownProperty(_PUSHDOWN_GROUPAGGS) {
		this.resetIndexGroupAggs()
	}

	// build plan for aggregates
	indexGroupAggs, indexProjection = this.buildIndexGroupAggs(entry, keys, unnestScan, indexProjection)
	projDistinct := entry.IsPushDownProperty(_PUSHDOWN_DISTINCT)

	cost, cardinality, size, frCost := OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
	if useCBO && entry.cost > 0.0 && entry.cardinality > 0.0 && entry.size > 0 && entry.frCost > 0.0 {
		if indexGroupAggs != nil {
			cost, cardinality, size, frCost = getIndexGroupAggsCost(index, indexGroupAggs,
				indexProjection, this.keyspaceNames, entry.cardinality)
		} else {
			cost, cardinality, size, frCost = getIndexProjectionCost(index, indexProjection, entry.cardinality)
		}

		if cost > 0.0 && cardinality > 0.0 && size > 0 && frCost > 0.0 {
			entry.cost += cost
			entry.cardinality = cardinality
			entry.size += size
			entry.frCost += frCost
		}
	}

	arrayKey := entry.arrayKey
	if !implicitAny {
		arrayKey = nil
	}

	// generate filters for covering index scan
	// do not generate filter in case of primary scan used on inner of nested-loop join or
	// in correlated subquery (cache used in IndexScan3)
	var filter expression.Expression
	if indexGroupAggs == nil && (len(this.baseKeyspaces) > 1 || implicitAny || len(entry.includes) > 0) &&
		!(index.IsPrimary() && (this.hasBuilderFlag(BUILDER_JOIN_ON_PRIMARY) || node.IsInCorrSubq())) {

		var err error
		var unnestAliases []string
		if unnestScan {
			unnestAliases = entry.unnestAliases
		}
		filter, cost, cardinality, size, frCost, err = this.getIndexFilter(index, node.Alias(), entry.spans, entry.includeSpans,
			arrayKey, unnestAliases, covers, filterCovers, entry.cost, entry.cardinality, entry.size, entry.frCost)
		if err != nil {
			return nil, 0, err
		}
		if useCBO {
			entry.cardinality, entry.cost, entry.frCost, entry.size = cardinality, cost, frCost, size
		}
	}

	var indexKeyNames []string
	var indexPartitionSets plan.IndexPartitionSets
	if index6, ok := entry.index.(datastore.Index6); ok && index6.IsBhive() && entry.HasFlag(IE_VECTOR_KEY_SARGABLE) {
		var err error
		if filter != nil {
			indexKeyNames, err = getIndexKeyNames(node.Alias(), index, indexProjection, true)
			if err != nil {
				return nil, 0, err
			}
		}
		indexPartitionSets, err = this.getIndexPartitionSets(entry.partitionKeys, node, pred, baseKeyspace)
		if err != nil {
			return nil, 0, err
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

	if indexGroupAggs != nil {
		indexKeyCovers := make(expression.Covers, len(covers))
		k := 0
		emptyCover := expression.NewCover(expression.EMPTY_STRING_EXPR)
		for i := 0; i < len(covers); i++ {
			if k >= len(indexGroupAggs.DependsOnIndexKeys) || i != indexGroupAggs.DependsOnIndexKeys[k] {
				indexKeyCovers[i] = emptyCover
			} else {
				indexKeyCovers[i] = covers[i]
				k++
			}

		}
		covers = indexKeyCovers
	}

	// build plan for IndexScan
	scan = entry.spans.CreateScan(index, node, this.context.IndexApiVersion(), false, projDistinct,
		overlapSpans(pred), array, this.offset, this.limit, indexProjection, indexKeyOrders,
		indexGroupAggs, covers, filterCovers, filter, entry.cost, entry.cardinality,
		entry.size, entry.frCost, includeSpans, baseKeyspace, hasDeltaKeyspace, skipNewKeys,
		this.hasBuilderFlag(BUILDER_NL_INNER), false, indexKeyNames, indexPartitionSets)
	if scan != nil {
		scan.SetImplicitArrayKey(arrayKey)
		if entry.index.Type() != datastore.SYSTEM {
			this.collectIndexKeyspaceNames(baseKeyspace.Keyspace())
		}
		this.coveringScans = append(this.coveringScans, scan)
	}

	return scan, sargLength, nil
}

func (this *builder) checkResetPaginations(entry *indexEntry,
	keys datastore.IndexKeys, id expression.Expression) (indexKeyOrders plan.IndexKeyOrders) {

	// check order pushdown and reset
	if this.order != nil {
		if entry.IsPushDownProperty(_PUSHDOWN_ORDER) || entry.IsPushDownProperty(_PUSHDOWN_PARTIAL_ORDER) {
			_, indexKeyOrders, this.partialSortTermCount = this.useIndexOrder(entry, keys, id, entry.pushDownProperty)
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
	scan := entry.spans.CreateScan(entry.index, node, this.context.IndexApiVersion(), false, false, overlapSpans(pred),
		array, nil, expression.ONE_EXPR, indexProjection, indexKeyOrders, nil, covers, filterCovers, nil,
		OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL, nil, baseKeyspace,
		false, false, this.hasBuilderFlag(BUILDER_NL_INNER), false, nil, nil)
	if scan != nil {
		if entry.index.Type() != datastore.SYSTEM {
			this.collectIndexKeyspaceNames(baseKeyspace.Keyspace())
		}
		this.coveringScans = append(this.coveringScans, scan)
	}

	return scan
}

func (this *builder) getExprsToCover() expression.Expressions {
	exprs := this.cover.Expressions()
	if this.where == nil || !this.hasBuilderFlag(BUILDER_WHERE_DEPENDS_ON_LET) {
		return exprs
	}
	newExprs := make(expression.Expressions, 0, len(exprs)+1)
	newExprs = append(newExprs, exprs...)
	newExprs = append(newExprs, this.where)
	return newExprs
}

func mapFilterCovers(fc map[expression.Expression]value.Value, fullCover bool) map[*expression.Cover]value.Value {
	if len(fc) == 0 {
		return nil
	}

	fsc := make(map[string]bool, len(fc))
	rv := make(map[*expression.Cover]value.Value, len(fc))
	for e, v := range fc {
		str := e.String()
		if _, ok := fsc[str]; ok {
			continue
		}
		var c *expression.Cover
		if fullCover {
			c = expression.NewCover(e)
		} else {
			c = expression.NewIndexCondition(e)
		}
		rv[c] = v
		fsc[str] = true
	}

	return rv
}

func indexCoverExpressions(entry *indexEntry, keys datastore.IndexKeys, inclInclude bool,
	pred, origPred expression.Expression, keyspace string, context *PrepareContext) (
	expression.Expressions, map[*expression.Cover]value.Value, error) {

	var filterCovers map[*expression.Cover]value.Value
	flatten := entry.arrayKey != nil && entry.arrayKey.Flatten()
	vector := entry.HasFlag(IE_VECTOR_KEY_SARGABLE)

	var vecExpr *expression.ApproxVectorDistance
	if vector {
		if tspans, ok := entry.spans.(*TermSpans); ok {
			vecExpr = tspans.vecExpr
		}
		if vecExpr == nil {
			return nil, nil, errors.NewPlanInternalError("indexCoverExpressions: vector search predicate not available")
		}

		if vecExpr.HasReRank(true) {
			index6, ok := entry.index.(datastore.Index6)
			if !ok {
				return nil, nil, errors.NewPlanInternalError("indexCoverExpressions: vector search index not index6")
			}
			if !index6.IsBhive() || !index6.AllowRerank() {
				// if ReRank is specified but unknown, or it's true, cannot cover
				// exception: BHive index that allows reranking can cover
				vecExpr = nil
			}
		}
	}

	size := len(keys)
	if inclInclude {
		size += len(entry.includes)
	}
	exprs := make(expression.Expressions, 0, size)
	for _, key := range keys {
		if key.HasAttribute(datastore.IK_VECTOR) {
			// only put any covered vector expression; do not put the index key here since
			// that may cover other expressions involving the same field/expression
			// incorrectly
			if vector && vecExpr != nil {
				exprs = append(exprs, vecExpr)
			}
		} else if _, ok := key.Expr.(*expression.All); ok && flatten {
			exprs = append(exprs, entry.arrayKey)
		} else {
			exprs = append(exprs, key.Expr)
		}
	}
	if inclInclude && len(entry.includes) > 0 {
		exprs = append(exprs, entry.includes...)
	}

	if entry.cond != nil {
		fc := make(map[expression.Expression]value.Value, 2)
		fc = entry.cond.FilterExpressionCovers(fc)
		fc = entry.origCond.FilterExpressionCovers(fc)
		filterCovers = mapFilterCovers(fc, true)
	}

	// Allow array indexes to cover ANY predicates
	if pred != nil && entry.hasExactSpans() && implicitAnyCover(entry, false, uint64(0)) {
		covers, err := CoversFor(pred, origPred, keys, context)
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

	if len(filterCovers) > 0 {
		for c, _ := range filterCovers {
			exprs = append(exprs, c.Covered())
		}
	}

	return exprs, filterCovers, nil
}

func hasSargableArrayKey(entry *indexEntry) bool {
	if entry.arrayKey != nil {
		for i, k := range entry.sargKeys {
			if _, ok := k.(*expression.All); ok &&
				i < len(entry.skeys) && entry.skeys[i] {
				return true
			}
		}
	}
	return false
}

func hasUnknownsInSargableArrayKey(entry *indexEntry) bool {
	if entry.arrayKey == nil || entry.spans == nil {
		return false
	}
	cnt := 0
	size := 1
	if entry.arrayKey.Flatten() {
		size = entry.arrayKey.FlattenSize()
	}
	for i, _ := range entry.sargKeys {
		if i >= entry.arrayKeyPos && i < entry.arrayKeyPos+size &&
			i < len(entry.skeys) && entry.skeys[i] {
			cnt++
			if !entry.spans.CanProduceUnknowns(i) {
				return false
			}
		}
	}
	return cnt > 0 || (entry.arrayKeyPos == 0)
}

func implicitFilterCovers(expr expression.Expression) map[*expression.Cover]value.Value {
	var fc map[expression.Expression]value.Value
	for all, ok := expr.(*expression.All); ok; all, ok = expr.(*expression.All) {
		if array, ok := all.Array().(*expression.Array); ok {
			if fc == nil {
				fc = make(map[expression.Expression]value.Value, len(array.Bindings())+1)
			}
			for _, b := range array.Bindings() {
				fc[b.Expression()] = value.TRUE_ARRAY_VALUE
			}
			if array.When() != nil {
				fc = array.When().FilterExpressionCovers(fc)
			}
			expr = array.ValueMapping()
		} else {
			break
		}
	}
	return mapFilterCovers(fc, true)
}

func replaceFlattenKeys(keys datastore.IndexKeys, entry *indexEntry) (rv datastore.IndexKeys) {
	all := entry.arrayKey
	pos := entry.arrayKeyPos
	if all == nil || !all.Flatten() {
		return keys
	}
	rv = make(datastore.IndexKeys, 0, len(keys))
	rv = append(rv, keys[0:pos]...)
	flattenKeys := all.FlattenKeys()
	for i, op := range flattenKeys.Operands() {
		rv = append(rv, &datastore.IndexKey{op, datastore.GetFlattenKeyAttributes(flattenKeys, i)})
	}
	rv = append(rv, keys[pos+all.FlattenSize():]...)
	return rv
}

func implicitIndexKeysProj(keys datastore.IndexKeys,
	anys map[expression.Expression]expression.Expression) (rv map[int]bool) {
	rv = make(map[int]bool, len(keys))
	for keyPos, indexKey := range keys {
		for _, expr := range anys {
			if expr.DependsOn(indexKey.Expr) {
				rv[keyPos] = true
				break
			}
		}
	}
	return
}

func implicitAnyCover(entry *indexEntry, flatten bool, featControl uint64) bool {
	_, ok := entry.spans.(*IntersectSpans)
	if ok || entry.arrayKey == nil || !hasSargableArrayKey(entry) || hasUnknownsInSargableArrayKey(entry) {
		return false
	}
	enabled := !flatten || (util.IsFeatureEnabled(featControl, util.N1QL_IMPLICIT_ARRAY_COVER) &&
		!bindingExpressionInIndexKeys(entry))
	return enabled && (flatten == entry.arrayKey.Flatten())
}

func bindingExpressionInIndexKeys(entry *indexEntry) bool {
	if entry.arrayKey == nil {
		return false
	}
	array, ok := entry.arrayKey.Array().(*expression.Array)
	if !ok {
		for _, key := range entry.keys {
			if expression.Equivalent(key, entry.arrayKey.Array()) {
				return true
			}
		}
		return false
	}
outer:
	for _, b := range array.Bindings() {
		for _, key := range entry.keys {
			if expression.Equivalent(key, b.Expression()) {
				continue outer
			}
		}
		return false
	}
	return true
}

func replaceVectorKey(keys datastore.IndexKeys, entry *indexEntry, cover bool) (datastore.IndexKeys, *expression.ApproxVectorDistance, error) {
	var vecExpr *expression.ApproxVectorDistance
	if tspans, ok := entry.spans.(*TermSpans); ok {
		vecExpr = tspans.vecExpr
	}
	if vecExpr == nil {
		return keys, nil, errors.NewPlanInternalError("replaceVectorKey: vector search predicate not available")
	}

	if vecExpr.HasReRank(true) && cover {
		index6, ok := entry.index.(datastore.Index6)
		if !ok {
			return keys, nil, errors.NewPlanInternalError("replaceVectorKey: vector search index not index6")
		}
		if !index6.IsBhive() || !index6.AllowRerank() {
			// if ReRank is specified but unknown, or it's true, cannot cover
			// exception: BHive index that allows reranking can cover
			return keys, nil, nil
		}
	}

	newKeys := make(datastore.IndexKeys, len(keys))
	for i := range keys {
		if keys[i].HasAttribute(datastore.IK_VECTOR) {
			newKeys[i] = &datastore.IndexKey{vecExpr, (keys[i].Attributes ^ datastore.IK_VECTOR)}
		} else {
			newKeys[i] = keys[i]
		}
	}

	return newKeys, vecExpr, nil
}

var _FILTER_COVERS_POOL = value.NewStringValuePool(32)
var _STRING_BOOL_POOL = util.NewStringBoolPool(1024)
