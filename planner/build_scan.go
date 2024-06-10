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
	"github.com/couchbase/query/value"
)

func (this *builder) selectScan(keyspace datastore.Keyspace, node *algebra.KeyspaceTerm,
	mutate bool) (op plan.Operator, err error) {

	keys := node.Keys()
	if keys != nil {
		if this.hasBuilderFlag(BUILDER_CHK_INDEX_ORDER) {
			return nil, nil
		}

		if !node.IsAnsiJoinOp() && this.falseWhereClause() {
			// WHERE clause is false, ignore the specified keys, use an empty array
			keys = expression.EMPTY_ARRAY_EXPR
		}

		this.resetPushDowns()
		switch keys.(type) {
		case *expression.ArrayConstruct, *algebra.NamedParameter, *algebra.PositionalParameter:
			this.maxParallelism = 0
		default:
			this.maxParallelism = 1
		}

		cost := OPT_COST_NOT_AVAIL
		cardinality := OPT_CARD_NOT_AVAIL
		size := OPT_SIZE_NOT_AVAIL
		frCost := OPT_COST_NOT_AVAIL
		if this.useCBO && this.keyspaceUseCBO(node.Alias()) {
			cost, cardinality, size, frCost = getKeyScanCost(keys)
		}
		return plan.NewKeyScan(keys, mutate, cost, cardinality, size, frCost), nil
	}

	secondary, primary, err := this.buildScan(keyspace, node)
	if err != nil {
		return nil, err
	}

	if !this.joinEnum() && !node.IsAnsiJoinOp() {
		err = this.markOptimHints(node.Alias())
		if err != nil {
			return nil, err
		}
	}

	if secondary != nil {
		return secondary, nil
	}
	if node.IsInCorrSubq() && !node.IsSystem() {
		return nil, errors.NewSubqueryMissingIndexError(node.Alias())
	}
	if primary != nil {
		return primary, nil
	}

	return nil, nil
}

func (this *builder) buildScan(keyspace datastore.Keyspace, node *algebra.KeyspaceTerm) (
	secondary plan.Operator, primary plan.Operator, err error) {

	join := node.IsAnsiJoinOp()
	if !join && this.falseWhereClause() {
		return _EMPTY_PLAN, nil, nil
	}

	baseKeyspace, ok := this.baseKeyspaces[node.Alias()]
	if !ok {
		return nil, nil, errors.NewPlanInternalError(fmt.Sprintf("buildScan: cannot find keyspace %s", node.Alias()))
	}

	var hints, virtualIndexes []datastore.Index
	if this.indexAdvisor {
		virtualIndexes = this.getIdxCandidates()
	}
	if len(node.Indexes()) > 0 || this.context.UseFts() {
		hints, err = allHints(keyspace, node.Indexes(), virtualIndexes, this.context.IndexApiVersion(), this.context.UseFts())
		if nil != hints {
			defer _INDEX_POOL.Put(hints)
		} else if len(node.Indexes()) > 0 {
			// if index hints are specified but none of the indexes are valid
			// mark index hint error
			baseKeyspace.SetIndexHintError()
		}
		if err != nil {
			return
		}
	}

	hasDeltaKeyspace := this.context.HasDeltaKeyspace(baseKeyspace.Keyspace())
	if hasDeltaKeyspace {
		this.resetPushDowns()
	}
	id := expression.NewField(
		expression.NewMeta(expression.NewIdentifier(node.Alias())),
		expression.NewFieldName("id", false))

	// for ANSI JOIN, the following process is already done for ON clause filters
	if !join && !this.hasBuilderFlag(BUILDER_CHK_INDEX_ORDER) {
		if !baseKeyspace.IsOuter() && len(baseKeyspace.JoinFilters()) > 0 {
			// derive IS NOT NULL predicate
			err = deriveNotNullFilter(keyspace, baseKeyspace, this.useCBO,
				this.context.IndexApiVersion(), virtualIndexes,
				this.advisorValidate(), this.context, this.aliases)
			if err != nil {
				return nil, nil, err
			}
		}

		// add predicates from UNNEST keyspaces
		err = addUnnestPreds(this.baseKeyspaces, baseKeyspace)
		if err != nil {
			return nil, nil, err
		}

		// include pushed ON-clause filter
		err = CombineFilters(baseKeyspace, true)
		if err != nil {
			return nil, nil, err
		}
	}

	this.collectPredicates(baseKeyspace, keyspace, node, nil, false, true)
	secondary, primary, err = this.buildPredicateScan(keyspace, node, baseKeyspace, id, hints, virtualIndexes)
	return secondary, primary, err
}

func (this *builder) buildPredicateScan(keyspace datastore.Keyspace, node *algebra.KeyspaceTerm,
	baseKeyspace *base.BaseKeyspace, id expression.Expression,
	hints, virtualIndexes []datastore.Index) (
	secondary plan.Operator, primary plan.Operator, err error) {

	// Handle constant FALSE predicate
	if baseKeyspace.OrigPred() != nil {
		cpred := baseKeyspace.OrigPred().Value()
		if cpred != nil && !cpred.Truth() {
			return _EMPTY_PLAN, nil, nil
		}
	}

	// do not consider primary index for ANSI JOIN or ANSI NEST
	var primaryKey expression.Expressions
	if !node.IsAnsiJoinOp() || node.IsUnderHash() || node.IsSystem() {
		primaryKey = expression.Expressions{id}
	}

	formalizer := expression.NewSelfFormalizer(node.Alias(), nil)

	if len(hints) > 0 {
		// Set processing HINT Indexes
		this.hintIndexes = true
		secondary, primary, err = this.buildSubsetScan(
			keyspace, node, baseKeyspace, id, hints, primaryKey, formalizer, true)
		this.hintIndexes = false
		if secondary != nil || primary != nil || err != nil {
			return
		}
		// no scan built with optimizer hints - mark index hint error
		baseKeyspace.SetIndexHintError()
	}

	// collect SEARCH() functions that depends on current keyspace in the predicate
	var searchFns map[string]*search.Search
	if !node.IsUnderNL() {
		pred := baseKeyspace.DnfPred()
		if node.IsAnsiJoinOp() && baseKeyspace.OnclauseOnly() {
			pred = baseKeyspace.Onclause()
		}

		searchFns = make(map[string]*search.Search)

		if err = collectFTSSearch(node.Alias(), searchFns, pred); err != nil {
			return
		}
	}

	others, err := allIndexes(keyspace, hints, virtualIndexes, this.context.IndexApiVersion(), len(searchFns) > 0)
	if nil != others {
		defer _INDEX_POOL.Put(others)
	}
	if err != nil {
		return
	}

	secondary, primary, err = this.buildSubsetScan(keyspace, node,
		baseKeyspace, id, others, primaryKey, formalizer, false)

	if secondary != nil || primary != nil || err != nil {
		return
	}

	if node.IsAnsiJoinOp() {
		if node.IsPrimaryJoin() || node.IsUnderHash() || node.IsSystem() {
			return nil, nil, nil
		} else {
			op := "join"
			if node.IsAnsiNest() {
				op = "nest"
			}
			return nil, nil, errors.NewNoAnsiJoinError(node.Alias(), op)
		}
	} else if this.hasBuilderFlag(BUILDER_CHK_INDEX_ORDER) {
		return nil, nil, nil
	} else {
		return nil, nil, errors.NewPlanInternalError(fmt.Sprintf("buildPredicateScan: No plan generated for %s", node.Alias()))
	}
}

func (this *builder) buildSubsetScan(keyspace datastore.Keyspace, node *algebra.KeyspaceTerm,
	baseKeyspace *base.BaseKeyspace, id expression.Expression, indexes []datastore.Index,
	primaryKey expression.Expressions, formalizer *expression.Formalizer, force bool) (
	secondary plan.Operator, primary plan.Operator, err error) {

	join := node.IsAnsiJoinOp()
	hash := node.IsUnderHash()
	if join {
		this.resetPushDowns()
	}
	order := this.order
	offset := this.offset
	limit := this.limit

	pred := baseKeyspace.DnfPred()
	if join && baseKeyspace.OnclauseOnly() {
		pred = baseKeyspace.Onclause()
	}
	if !this.hasBuilderFlag(BUILDER_CHK_INDEX_ORDER) {
		// Prefer OR scan
		if or, ok := pred.(*expression.Or); ok {

			scan, _, err := this.buildOrScan(node, baseKeyspace, id, or, indexes, primaryKey, formalizer)

			if scan != nil || err != nil {
				return scan, nil, err
			}
		}
	}

	// Prefer secondary scan
	secondary, _, err = this.buildTermScan(node, baseKeyspace, id, indexes, primaryKey, formalizer)
	if secondary != nil || err != nil {
		return secondary, nil, err
	}

	if !join || hash || node.IsSystem() {
		// No secondary scan, try primary scan. restore order there is predicate no need to restore others
		this.order = order
		exact := false
		hasDeltaKeyspace := this.context.HasDeltaKeyspace(baseKeyspace.Keyspace())
		if pred == nil && !hash && !hasDeltaKeyspace {
			this.offset = offset
			this.limit = limit
			exact = true
		}
		primary, err = this.buildPrimaryScan(keyspace, node, indexes, id, force, exact, hasDeltaKeyspace)
		if this.hasBuilderFlag(BUILDER_CHK_INDEX_ORDER) && order != nil && this.order == nil {
			// building ORDER plan during join enumeration and primary scan does not have order
			return nil, nil, nil
		}
	}
	return nil, primary, err
}

func (this *builder) buildTermScan(node *algebra.KeyspaceTerm,
	baseKeyspace *base.BaseKeyspace, id expression.Expression, indexes []datastore.Index,
	primaryKey expression.Expressions, formalizer *expression.Formalizer) (
	secondary plan.SecondaryScan, sargLength int, err error) {

	join := node.IsAnsiJoinOp()

	var scanbuf [4]plan.SecondaryScan
	scans := scanbuf[0:1]

	if !join && !this.hasBuilderFlag(BUILDER_CHK_INDEX_ORDER) {
		// Consider pattern matching indexes
		err = this.PatternFor(baseKeyspace, indexes, formalizer)
		if err != nil {
			return nil, 0, err
		}
	}

	pred := baseKeyspace.DnfPred()
	if join && baseKeyspace.OnclauseOnly() {
		pred = baseKeyspace.Onclause()
	}

	subset := pred
	if len(this.context.NamedArgs()) > 0 || len(this.context.PositionalArgs()) > 0 {
		subset, err = base.ReplaceParameters(subset, this.context.NamedArgs(), this.context.PositionalArgs())
		if err != nil {
			return
		}
	}

	// collect UNNEST bindings when HINT indexes has FTS index
	var ubs expression.Bindings
	if !this.hasBuilderFlag(BUILDER_CHK_INDEX_ORDER) {
		if this.hintIndexes && this.from != nil {
			for _, idx := range indexes {
				if idx.Type() == datastore.FTS {
					ubs = make(expression.Bindings, 0, 2)
					ua := expression.Expressions{expression.NewIdentifier(node.Alias())}
					_, ubs = this.collectUnnestBindings(this.from, ua, ubs)
					break
				}
			}
		}
	}

	sargables, arrays, flex, err := this.sargableIndexes(indexes, pred, subset, primaryKey,
		formalizer, ubs, node.IsUnderNL())
	if err != nil {
		return nil, 0, err
	}

	if this.hasBuilderFlag(BUILDER_CHK_INDEX_ORDER) {
		// useIndexOrder() needs index span
		err = this.sargIndexes(baseKeyspace, false, sargables)
		if err != nil {
			return nil, 0, err
		}
		// only consider indexes that satisfy ordering
		for i, e := range sargables {
			ok, _ := this.useIndexOrder(e, e.keys)
			if !ok {
				delete(sargables, i)
			}
		}
		if len(sargables) == 0 {
			return nil, 0, nil
		}
	}

	// purge any subset indexe and keep superset indexes
	minimals := this.minimalIndexes(sargables, false, pred, node)
	flex = this.minimalFTSFlexIndexes(flex, false)

	// pred has SEARCH() function get sargable FTS indexes
	var searchSargables []*indexEntry
	var searchFns map[string]*search.Search
	if !node.IsUnderNL() {
		searchFns = make(map[string]*search.Search)
		if err = collectFTSSearch(node.Alias(), searchFns, pred); err != nil {
			return nil, 0, err
		}

		searchSargables, err = this.sargableSearchIndexes(indexes, pred, searchFns, formalizer)
		if err != nil {
			return nil, 0, err
		}
	}

	indexPushDowns := this.storeIndexPushDowns()
	this.orderScan = nil

	defer func() {
		if this.orderScan != nil {
			this.order = indexPushDowns.order
		}
	}()

	var secOffsetPushed, unnestOffsetPushed, dynamicOffsetPushed bool
	var limitPushed bool

	// Try secondary scan
	if len(minimals) > 0 || len(arrays) > 0 || len(searchSargables) > 0 || len(flex) > 0 {
		secondary, sargLength, err = this.buildSecondaryScan(minimals, arrays, flex, node, baseKeyspace,
			subset, id, searchSargables)
		if err != nil {
			return nil, 0, err
		}

		if secondary != nil {
			if len(this.coveringScans) > 0 || this.countScan != nil {
				return secondary, sargLength, nil
			}

			if secondary == this.orderScan {
				scans[0] = secondary
			} else {
				scans = append(scans, secondary)
			}

			secOffsetPushed = this.offset != nil
			limitPushed = limitPushed || this.limit != nil
		}
	}

	// Try dynamic scan
	if !join && len(arrays) > 0 && baseKeyspace.OrigPred() != nil {
		// Try pushdowns
		if indexPushDowns.order == nil || this.orderScan != nil {
			this.limit = indexPushDowns.limit
			this.offset = indexPushDowns.offset
		}

		dynamicPred := baseKeyspace.OrigPred().Copy()
		dnf := base.NewDNF(dynamicPred, false, true)
		dynamicPred, err = dnf.Map(dynamicPred)
		if err != nil {
			return nil, 0, err
		}

		dynamic, dynamicSargLength, err :=
			this.buildDynamicScan(node, id, dynamicPred, arrays, primaryKey, formalizer)
		if err != nil {
			return nil, 0, err
		}

		if dynamic != nil {
			if len(this.coveringScans) > 0 || this.countScan != nil {
				return dynamic, dynamicSargLength, err
			}

			scans = append(scans, dynamic)
			if sargLength < dynamicSargLength {
				sargLength = dynamicSargLength
			}
			dynamicOffsetPushed = this.offset != nil
			limitPushed = limitPushed || this.limit != nil
		}
	}

	switch len(scans) {
	case 0:
		this.limit = indexPushDowns.limit
		this.offset = indexPushDowns.offset
		secondary = nil
	case 1:
		this.resetOffset()
		if secOffsetPushed || unnestOffsetPushed || dynamicOffsetPushed {
			this.offset = indexPushDowns.offset
		}
		secondary = scans[0]
	default:
		this.resetOffset()
		var limit expression.Expression

		if limitPushed {
			limit = offsetPlusLimit(indexPushDowns.offset, indexPushDowns.limit)
		}

		if scans[0] == nil {
			if len(scans) == 2 {
				if secOffsetPushed || unnestOffsetPushed || dynamicOffsetPushed {
					this.offset = indexPushDowns.offset
				}
				secondary = scans[1]
			} else {
				cost, cardinality, size, frCost := this.intersectScanCost(node, scans[1:]...)
				secondary = plan.NewIntersectScan(limit, cost, cardinality, size, frCost, scans[1:]...)
			}
		} else {
			if ordered, ok := scans[0].(*plan.OrderedIntersectScan); ok {
				scans = append(ordered.Scans(), scans[1:]...)
			}

			cost, cardinality, size, frCost := this.intersectScanCost(node, scans...)
			secondary = plan.NewOrderedIntersectScan(nil, cost, cardinality, size, frCost, scans...)
		}
	}

	// Return secondary scan, if any
	return secondary, sargLength, nil
}

func (this *builder) processPredicate(pred expression.Expression, isOnclause bool) (
	value.Value, error) {

	return this.processPredicateBase(pred, this.baseKeyspaces, isOnclause)
}

func (this *builder) processPredicateBase(pred expression.Expression,
	baseKeyspaces map[string]*base.BaseKeyspace, isOnclause bool) (value.Value, error) {

	var err error
	this.arrayId, err = expression.AssignArrayId(pred, this.arrayId)
	if err != nil {
		return nil, err
	}

	return ClassifyExpr(pred, baseKeyspaces, this.keyspaceNames, isOnclause, this.useCBO,
		this.advisorValidate(), this.context)
}

func (this *builder) processWhere(where expression.Expression) (err error) {
	var constant value.Value
	constant, err = this.processPredicate(where, false)
	if err != nil {
		return
	}
	// Handle constant TRUE/FALSE predicate
	if constant != nil {
		if constant.Truth() {
			this.setTrueWhereClause()
		} else {
			this.setFalseWhereClause()
		}
	}

	return
}

func (this *builder) intersectScanCost(node *algebra.KeyspaceTerm, scans ...plan.SecondaryScan) (
	float64, float64, int64, float64) {
	useCBO := this.useCBO
	if !useCBO {
		return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
	}

	cnt := this.getDocCount(node.Alias())
	if cnt < 0 {
		return OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
	} else if cnt == 0 {
		// empty keyspace, use 1 instead to avoid divide by 0
		cnt = 1
	}
	docCount := float64(cnt)

	var cost, cardinality, frCost, selec float64
	var size int64
	for i, scan := range scans {
		scost := scan.Cost()
		scardinality := scan.Cardinality()
		ssize := scan.Size()
		sfrCost := scan.FrCost()
		if (scost <= 0.0) || (scardinality <= 0.0) || (ssize <= 0) || (sfrCost <= 0.0) {
			useCBO = false
			break
		}

		selec1 := scardinality / docCount
		if selec1 > 1.0 {
			selec1 = 1.0
		}
		if i == 0 {
			selec = selec1
			cost = scost
			frCost = sfrCost
			size = ssize
		} else {
			selec = selec * selec1
			// index scans under intersect scan execute in parallel at runtime,
			// thus we take the one with the highest cost as cost instead of adding
			// all cost, this should be more reflective of execution time of an
			// intersect scan
			if scost > cost {
				cost = scost
				frCost = sfrCost
				size = ssize
			}
		}
	}

	if useCBO {
		// cost calculated in for loop above
		cardinality = selec * docCount
	} else {
		cost = OPT_COST_NOT_AVAIL
		cardinality = OPT_CARD_NOT_AVAIL
		size = OPT_SIZE_NOT_AVAIL
		frCost = OPT_COST_NOT_AVAIL
	}

	return cost, cardinality, size, frCost
}

// helper function check online indexes
func isValidIndex(idx datastore.Index, indexApiVersion int) bool {
	state, _, err := idx.State()
	if err != nil {
		logging.Errorf("Index selection error: %v", err.Error())
		return false
	}

	if idx.Type() == datastore.FTS {
		return state == datastore.ONLINE
	}

	return (state == datastore.ONLINE) && (useIndex2API(idx, indexApiVersion) || !indexHasDesc(idx))
}

func poolAllocIndexSlice(indexes []datastore.Index) []datastore.Index {
	if nil == indexes {
		indexes = _INDEX_POOL.Get()
	}
	return indexes
}

// all HINT indexes
func allHints(keyspace datastore.Keyspace, hints algebra.IndexRefs, virtualIndexes []datastore.Index, indexApiVersion int, useFts bool) (
	[]datastore.Index, error) {

	var indexes []datastore.Index
	// check if HINT has FTS index refrence
	var hintFts bool

	for _, hint := range hints {
		if hint.Using() == datastore.FTS {
			hintFts = true
			break
		}
	}

	indexers, err := keyspace.Indexers()
	if err != nil {
		return nil, err
	}

	for _, indexer := range indexers {
		// neither FTS index reference in the HINT nor useFts set skip FTS indexer
		if !hintFts && !useFts && indexer.Name() == datastore.FTS {
			continue
		}

		idxes, err := indexer.Indexes()
		if err != nil {
			return indexes, err
		}

		// all HINT indexes. If name is "", consider all indexes on the indexer
		// duplicates on the HINT will be ignored
		for _, idx := range idxes {
			/* When one or more FTS indexes is specified in the USE INDEX hint,
			   USE_FTS query parameter does not take effect. When no FTS indexes is specified in the
			   USE INDEX hint (or no hint specified), USE_FTS query parameter takes effect.
			*/
			if !hintFts && useFts && indexer.Name() == datastore.FTS && isValidIndex(idx, indexApiVersion) {
				indexes = append(poolAllocIndexSlice(indexes), idx)
				continue
			}

			for _, hint := range hints {
				using := hint.Using()
				if using == datastore.DEFAULT {
					using = datastore.GSI
				}
				if indexer.Name() == using &&
					(hint.Name() == "" || hint.Name() == idx.Name()) {
					if isValidIndex(idx, indexApiVersion) {
						indexes = append(poolAllocIndexSlice(indexes), idx)
					}
					break
				}
			}
		}
	}

	if len(virtualIndexes) > 0 {
		indexes = append(poolAllocIndexSlice(indexes), virtualIndexes...)
	}

	return indexes, nil
}

/*
all the indexes excluding HINT indexes.
inclFts indicates to include FTS index or not
        * true  - SEARCH() function is present
        * false - right side of some JOINs, no SERACH() function
*/

func allIndexes(keyspace datastore.Keyspace, skip, virtualIndexes []datastore.Index, indexApiVersion int, inclFts bool) (
	[]datastore.Index, error) {

	var indexes []datastore.Index

	indexers, err := keyspace.Indexers()
	if err != nil {
		return nil, err
	}

	var skipMap map[datastore.Index]bool
	if len(skip) > 0 {
		skipMap = _SKIP_POOL.Get()
		defer _SKIP_POOL.Put(skipMap)
		for _, s := range skip {
			skipMap[s] = true
		}
	}

	for _, indexer := range indexers {
		// no FTS indexes needed and  indexer is FTS skip the indexer
		if !inclFts && indexer.Name() == datastore.FTS {
			continue
		}

		idxes, err := indexer.Indexes()
		if err != nil {
			return indexes, err
		}

		for _, idx := range idxes {
			// Skip index if listed
			if len(skipMap) > 0 && skipMap[idx] {
				continue
			}

			if isValidIndex(idx, indexApiVersion) {
				indexes = append(poolAllocIndexSlice(indexes), idx)
			}

		}
	}

	for _, idx := range virtualIndexes {
		if len(skipMap) > 0 && skipMap[idx] {
			continue
		}
		indexes = append(poolAllocIndexSlice(indexes), idx)
	}

	return indexes, nil
}

func checkSubset(pred, cond expression.Expression, context *PrepareContext) bool {
	if context != nil && (len(context.NamedArgs()) > 0 || len(context.PositionalArgs()) > 0) {
		var err error
		pred, err = base.ReplaceParameters(pred, context.NamedArgs(), context.PositionalArgs())
		if err != nil {
			return false
		}
	}
	return base.SubsetOf(pred, cond)
}

var _INDEX_POOL = datastore.NewIndexPool(256)
var _SKIP_POOL = datastore.NewIndexBoolPool(32)
var _EMPTY_PLAN = plan.NewValueScan(algebra.Pairs{}, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL)
