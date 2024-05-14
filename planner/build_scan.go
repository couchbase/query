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
	"time"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/search"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/util"
)

func (this *builder) selectScan(keyspace datastore.Keyspace, node *algebra.KeyspaceTerm,
	mutate bool) (op plan.Operator, err error) {

	keys := node.Keys()
	alias := node.Alias()
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
		if this.useCBO && this.keyspaceUseCBO(alias) {
			cost, cardinality, size, frCost = getKeyScanCost(keys)
		}
		return plan.NewKeyScan(keys, mutate, cost, cardinality, size, frCost), nil
	}

	var primary, secondary plan.Operator
	secondary, primary, err = this.buildScan(keyspace, node)
	if err != nil {
		return nil, err
	}

	if !this.joinEnum() && !node.IsAnsiJoinOp() {
		if !node.HasTransferJoinHint() {
			baseKeyspace, _ := this.baseKeyspaces[alias]
			baseKeyspace.MarkJoinHintError(algebra.JOIN_HINT_FIRST_TERM + alias)
		}
		err = this.markOptimHints(alias, false)
		if err != nil {
			return nil, err
		}
	}

	if secondary != nil {
		return secondary, nil
	}

	if node.IsInCorrSubq() && !node.IsSystem() {
		if primary != nil {
			// early exit if using primary index scan/seq_scan for corrsubq
			if this.getDocCount(node.Alias()) > _MAX_PRIMARY_INDEX_CACHE_SIZE {
				return nil, errors.NewSubqueryNumDocsExceeded(node.Alias(), _MAX_PRIMARY_INDEX_CACHE_SIZE)
			}
		} else {
			return nil, errors.NewSubqueryMissingIndexError(node.Alias())
		}

	}

	return primary, nil
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
	if len(baseKeyspace.IndexHints()) > 0 || this.context.UseFts() {
		hints, err = allHints(keyspace, baseKeyspace.IndexHints(), virtualIndexes, this.context.IndexApiVersion(),
			this.context.UseFts(), util.IsFeatureEnabled(this.context.FeatureControls(), util.N1QL_SEQ_SCAN))
		if nil != hints {
			defer _INDEX_POOL.Put(hints)
		} else if len(baseKeyspace.IndexHints()) > 0 {
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
	if !join && !this.hasBuilderFlag(BUILDER_CHK_INDEX_ORDER|BUILDER_DO_JOIN_FILTER) {
		if len(baseKeyspace.JoinFilters()) > 0 {
			// derive IS NOT NULL predicate
			var duration time.Duration
			err, duration = deriveNotNullFilter(keyspace, baseKeyspace, this.useCBO,
				this.context.IndexApiVersion(), virtualIndexes,
				this.advisorValidate(), this.context, this.aliases,
				util.IsFeatureEnabled(this.context.FeatureControls(), util.N1QL_SEQ_SCAN))
			this.recordSubTime("index.metadata", duration)
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
	if err == nil && !join && this.useCBO && this.keyspaceUseCBO(node.Alias()) &&
		secondary != nil && len(baseKeyspace.GetUnnestIndexes()) > 0 {
		chkOpUnnestIndexes(secondary, baseKeyspace.GetUnnestIndexes(), nil)
	}
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
	nlPrimaryScan := !util.IsFeatureEnabled(this.context.FeatureControls(),
		util.N1QL_NL_PRIMARYSCAN) || this.hasBuilderFlag(BUILDER_JOIN_ON_PRIMARY)
	var primaryKey expression.Expressions
	if !node.IsAnsiJoinOp() || this.hasBuilderFlag(BUILDER_UNDER_HASH) || node.IsSystem() || nlPrimaryScan {
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
		if baseKeyspace.HasIndexAllHint() {
			// if INDEX_ALL hint is specified and not followed, consider all indexes
			hints = nil
		}
	}

	// collect SEARCH() functions that depends on current keyspace in the predicate
	var searchFns map[string]*search.Search
	if !this.hasBuilderFlag(BUILDER_NL_INNER) {
		pred := baseKeyspace.DnfPred()
		if node.IsAnsiJoinOp() && baseKeyspace.OnclauseOnly() {
			pred = baseKeyspace.Onclause()
		}

		searchFns = make(map[string]*search.Search)

		if err = collectFTSSearch(node.Alias(), searchFns, pred); err != nil {
			return
		}
	}

	others, err, duration := allIndexes(keyspace, hints, virtualIndexes, this.context.IndexApiVersion(), len(searchFns) > 0,
		util.IsFeatureEnabled(this.context.FeatureControls(), util.N1QL_SEQ_SCAN))
	if nil != others {
		defer _INDEX_POOL.Put(others)
	}
	this.recordSubTime("index.metadata", duration)
	if err != nil {
		return
	}

	secondary, primary, err = this.buildSubsetScan(keyspace, node,
		baseKeyspace, id, others, primaryKey, formalizer, false)

	if secondary != nil || primary != nil || err != nil {
		return
	}

	if node.IsAnsiJoinOp() {
		if node.IsPrimaryJoin() || this.hasBuilderFlag(BUILDER_UNDER_HASH) || node.IsSystem() || nlPrimaryScan {
			return nil, nil, nil
		} else {
			op := "join"
			if node.IsAnsiNest() {
				op = "nest"
			}
			return nil, nil, errors.NewNoAnsiJoinError(node.Alias(), op)
		}
	} else if this.hasBuilderFlag(BUILDER_CHK_INDEX_ORDER | BUILDER_DO_JOIN_FILTER) {
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
	hash := this.hasBuilderFlag(BUILDER_UNDER_HASH)
	nlPrimaryScan := !util.IsFeatureEnabled(this.context.FeatureControls(), util.N1QL_NL_PRIMARYSCAN) ||
		this.hasBuilderFlag(BUILDER_JOIN_ON_PRIMARY)
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
	if !this.hasBuilderFlag(BUILDER_CHK_INDEX_ORDER | BUILDER_JOIN_ON_PRIMARY) {
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

	if this.hasBuilderFlag(BUILDER_DO_JOIN_FILTER) {
		return nil, nil, nil
	}

	if !join || hash || node.IsSystem() || nlPrimaryScan {
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

	if !join && !this.hasBuilderFlag(BUILDER_CHK_INDEX_ORDER|BUILDER_DO_JOIN_FILTER|BUILDER_JOIN_ON_PRIMARY) {
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
	if !this.hasBuilderFlag(BUILDER_CHK_INDEX_ORDER | BUILDER_DO_JOIN_FILTER | BUILDER_JOIN_ON_PRIMARY) {
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
		formalizer, ubs, this.hasBuilderFlag(BUILDER_NL_INNER))
	if err != nil {
		return nil, 0, err
	}

	indexAll := this.hintIndexes && baseKeyspace.HasIndexAllHint()
	if indexAll {
		if this.hasBuilderFlag(BUILDER_CHK_INDEX_ORDER | BUILDER_DO_JOIN_FILTER | BUILDER_JOIN_ON_PRIMARY) {
			return nil, 0, nil
		} else if !checkIndexAllSargable(baseKeyspace, sargables) {
			// INDEX_ALL hint specified, but not all specified indexes are sargable,
			return nil, 0, nil
		}
	}

	err = this.sargIndexes(baseKeyspace, this.hasBuilderFlag(BUILDER_UNDER_HASH), sargables)
	if err != nil {
		return nil, 0, err
	}

	if this.hasBuilderFlag(BUILDER_CHK_INDEX_ORDER) {
		// only consider indexes that satisfy ordering
		for i, e := range sargables {
			ok, _, _ := this.useIndexOrder(e, e.keys)
			if !ok {
				delete(sargables, i)
			}
		}
		if len(sargables) == 0 {
			return nil, 0, nil
		}
	}

	// purge any subset indexe and keep superset indexes
	minimals := sargables
	if !indexAll {
		minimals = this.minimalIndexes(sargables, false, pred, node)
	}
	flex = this.minimalFTSFlexIndexes(flex, false)

	// pred has SEARCH() function get sargable FTS indexes
	var searchSargables []*indexEntry
	var searchFns map[string]*search.Search
	if !this.hasBuilderFlag(BUILDER_NL_INNER | BUILDER_CHK_INDEX_ORDER | BUILDER_DO_JOIN_FILTER | BUILDER_JOIN_ON_PRIMARY) {
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
	if !join && !this.hasBuilderFlag(BUILDER_CHK_INDEX_ORDER|BUILDER_DO_JOIN_FILTER|BUILDER_JOIN_ON_PRIMARY) &&
		len(arrays) > 0 && baseKeyspace.OrigPred() != nil {
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
				secondary = plan.NewIntersectScan(limit, indexAll, cost, cardinality, size, frCost, scans[1:]...)
			}
		} else {
			if ordered, ok := scans[0].(*plan.OrderedIntersectScan); ok {
				scans = append(ordered.Scans(), scans[1:]...)
			}

			cost, cardinality, size, frCost := this.intersectScanCost(node, scans...)
			secondary = plan.NewOrderedIntersectScan(nil, indexAll, cost, cardinality, size, frCost, scans...)
		}
	}

	return secondary, sargLength, nil
}

func (this *builder) processPredicate(pred expression.Expression, isOnclause bool) (
	expression.Expression, error) {

	return this.processPredicateBase(pred, this.baseKeyspaces, isOnclause)
}

func (this *builder) processPredicateBase(pred expression.Expression,
	baseKeyspaces map[string]*base.BaseKeyspace, isOnclause bool) (expression.Expression, error) {

	var err error
	this.arrayId, err = expression.AssignArrayId(pred, this.arrayId)
	if err != nil {
		return nil, err
	}

	return ClassifyExpr(pred, baseKeyspaces, this.keyspaceNames, isOnclause, this.useCBO,
		this.advisorValidate(), this.context)
}

func (this *builder) processWhere(where expression.Expression) (err error) {
	if !this.falseWhereClause() {
		var extraExpr expression.Expression
		extraExpr, err = this.processPredicate(where, false)
		if err != nil {
			return
		}
		if extraExpr != nil && extraExpr.Value() == nil {
			this.setBuilderFlag(BUILDER_HAS_EXTRA_FLTR)
		}
	}

	return
}

func (this *builder) getWhere(whereExpr expression.Expression) (where expression.Expression, err error) {
	where, err = expression.RemoveConstants(whereExpr)
	if err != nil || where == nil {
		return
	}

	// Handle constant TRUE/FALSE predicate
	constant := where.Value()
	if constant != nil {
		if constant.Truth() {
			this.setTrueWhereClause()
		} else {
			this.setFalseWhereClause()
		}
		where = nil
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
func allHints(keyspace datastore.Keyspace, hints []algebra.OptimHint, virtualIndexes []datastore.Index, indexApiVersion int,
	useFts bool, inclSeqScan bool) ([]datastore.Index, error) {

	var indexes []datastore.Index
	// check if HINT has FTS index refrence
	var hintFts bool

	for _, hint := range hints {
		switch hint.(type) {
		case *algebra.HintFTSIndex, *algebra.HintNoFTSIndex:
			hintFts = true
		}
		if hintFts {
			break
		}
	}

	indexers, err := keyspace.Indexers()
	if err != nil {
		return nil, err
	}

	for _, indexer := range indexers {
		if !inclSeqScan && indexer.Name() == datastore.SEQ_SCAN {
			continue
		}
		indexerFts := false
		if indexer.Name() == datastore.FTS {
			indexerFts = true
		}

		// neither FTS index reference in the HINT nor useFts set skip FTS indexer
		if !hintFts && !useFts && indexerFts {
			continue
		}

		idxes, err := indexer.Indexes()
		if err != nil {
			return indexes, err
		}

		// all HINT indexes. If name is "", consider all indexes on the indexer
		// duplicates on the HINT will be ignored
		for _, idx := range idxes {
			if !isValidIndex(idx, indexApiVersion) {
				continue
			}

			/* When one or more FTS indexes is specified in the USE INDEX hint,
			   USE_FTS query parameter does not take effect. When no FTS indexes is specified in the
			   USE INDEX hint (or no hint specified), USE_FTS query parameter takes effect.
			*/
			if !hintFts && useFts && indexerFts {
				indexes = append(poolAllocIndexSlice(indexes), idx)
				continue
			}

			hasIndex := false
			hasNoIndex := false
			inIndex := false
			inNoIndex := false
			inIndexAll := false
			for _, hint := range hints {
				if hint.State() != algebra.HINT_STATE_UNKNOWN {
					continue
				}

				var hintIndexes []string
				var fts, negative, indexAll bool
				switch hint := hint.(type) {
				case *algebra.HintIndex:
					hintIndexes = hint.Indexes()
					hasIndex = true
				case *algebra.HintFTSIndex:
					hintIndexes = hint.Indexes()
					hasIndex = true
					fts = true
				case *algebra.HintNoIndex:
					hintIndexes = hint.Indexes()
					hasNoIndex = true
					negative = true
				case *algebra.HintNoFTSIndex:
					hintIndexes = hint.Indexes()
					hasNoIndex = true
					fts = true
					negative = true
				case *algebra.HintIndexAll:
					hintIndexes = hint.Indexes()
					indexAll = true
				}

				if (indexerFts && fts) || (!indexerFts && !fts) {
					if len(hintIndexes) == 0 {
						if negative {
							inNoIndex = true
						} else {
							inIndex = true
						}
					} else {
						for _, name := range hintIndexes {
							if name == idx.Name() {
								if indexAll {
									inIndexAll = true
								} else if negative {
									inNoIndex = true
								} else {
									inIndex = true
								}
							}
						}
					}
				}
			}

			use := false
			if inIndexAll {
				if inIndex || inNoIndex {
					// We should have checked for mixed index hints
					return nil, errors.NewPlanInternalError("allHints: mixed INDEX_ALL hints and other index hints")
				}
				use = true
			} else {
				if inIndex && inNoIndex {
					// We should have removed any index from NO_INDEX/NO_INDEX_FTS
					// hint that is also present in INDEX/INDEX_FTS hint.
					return nil, errors.NewPlanInternalError(fmt.Sprintf("allHints: unexpected index %s in both INDEX/INDEX_FTS "+
						"and NO_INDEX/NO_INDEX_FTS hints", idx.Name()))
				} else if inIndex {
					use = true
				} else if !hasIndex && hasNoIndex && !inNoIndex {
					use = true
				}
			}

			if use {
				indexes = append(poolAllocIndexSlice(indexes), idx)
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

func allIndexes(keyspace datastore.Keyspace, skip, virtualIndexes []datastore.Index, indexApiVersion int, inclFts bool,
	inclSeqScan bool) ([]datastore.Index, error, time.Duration) {

	var indexes []datastore.Index

	start := util.Now()
	indexers, err := keyspace.Indexers()
	if err != nil {
		return nil, err, 0
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
		} else if !inclSeqScan && indexer.Name() == datastore.SEQ_SCAN {
			continue
		}

		idxes, err := indexer.Indexes()
		if err != nil {
			return indexes, err, util.Since(start)
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

	return indexes, nil, util.Since(start)
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
