//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.
//
//go:build enterprise

package planner

import (
	"sort"

	"github.com/couchbase/query-ee/dictionary"
	"github.com/couchbase/query-ee/optutil"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/util"
)

func checkCostModel(featureControls uint64) {
	if util.IsFeatureEnabled(featureControls, util.N1QL_CBO_NEW) {
		optutil.SetNewCostModel()
	}
}

func optDocCount(keyspace string) int64 {
	docCount, _, _, _ := dictionary.GetKeyspaceInfo(keyspace)
	return docCount
}

func optFilterSelectivity(filter *base.Filter, advisorValidate bool, context *PrepareContext) {
	optutil.FilterSelectivity(filter, advisorValidate, context)
	return
}

func optExprSelec(keyspaces map[string]string, pred expression.Expression, advisorValidate bool,
	context *PrepareContext) (float64, float64) {
	sel, arrSel, _ := optutil.ExprSelec(keyspaces, pred, advisorValidate, 0, context)
	return sel, arrSel
}

func optDefInSelec(keyspace, key string, advisorValidate bool) float64 {
	return optutil.DefInSelec(keyspace, key, advisorValidate)
}

func optDefLikeSelec(keyspace, key string, advisorValidate bool) float64 {
	return optutil.DefLikeSelec(keyspace, key, advisorValidate)
}

func optMarkIndexFilters(keys expression.Expressions, spans plan.Spans2,
	condition, filter expression.Expression, unnestAliases []string, baseKeyspace *base.BaseKeyspace) {
	optutil.MarkIndexFilters(keys, spans, condition, filter, unnestAliases, baseKeyspace)
}

func optMinCost() float64 {
	return optutil.MinCost()
}

func optCheckRangeExprs(baseKeyspaces map[string]*base.BaseKeyspace, advisorValidate bool,
	context *PrepareContext) {
	optutil.CheckRangeExprs(baseKeyspaces, advisorValidate, context)
}

func primaryIndexScanCost(primary datastore.PrimaryIndex, requestId string, context *PrepareContext) (
	float64, float64, int64, float64) {
	return optutil.CalcPrimaryIndexScanCost(primary, requestId, context)
}

func indexScanCost(index datastore.Index, sargKeys expression.Expressions, requestId string,
	spans SargSpans, alias, keyspace string, advisorValidate bool, context *PrepareContext) (
	float64, float64, float64, int64, float64, error) {
	switch spans := spans.(type) {
	case *TermSpans:
		return optutil.CalcIndexScanCost(index, sargKeys, requestId, spans.spans, alias, advisorValidate, context)
	case *IntersectSpans:
		return intersectSpansCost(index, sargKeys, requestId, spans, alias, keyspace, advisorValidate, context)
	case *UnionSpans:
		return unionSpansCost(index, sargKeys, requestId, spans, alias, keyspace, advisorValidate, context)
	}

	return OPT_COST_NOT_AVAIL, OPT_SELEC_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL,
		errors.NewPlanInternalError("indexScanCost: unexpected span type")
}

func unionSpansCost(index datastore.Index, sargKeys expression.Expressions, requestId string,
	unionSpan *UnionSpans, alias, keyspace string, advisorValidate bool, context *PrepareContext) (
	float64, float64, float64, int64, float64, error) {

	var cost, sel, frCost, nrows float64
	var size int64
	for i, span := range unionSpan.spans {
		tcost, tsel, tcard, tsize, tfrCost, e := indexScanCost(index, sargKeys, requestId, span, alias, keyspace,
			advisorValidate, context)
		if e != nil {
			return tcost, tsel, tcard, tsize, tfrCost, e
		}
		cost += tcost
		tnrows := tcard / tsel
		if i == 0 {
			sel = tsel
			nrows = tnrows
			frCost = tfrCost
			size = tsize
		} else {
			tsel = tsel * (tnrows / nrows)
			sel = sel + tsel - (sel * tsel)
			if tsize > size {
				size = tsize
			}
		}
	}

	return cost, sel, (sel * nrows), size, frCost, nil
}

func intersectSpansCost(index datastore.Index, sargKeys expression.Expressions, requestId string,
	intersectSpan *IntersectSpans, alias, keyspace string, advisorValidate bool, context *PrepareContext) (
	float64, float64, float64, int64, float64, error) {

	spanMap := make(map[*base.IndexCost]SargSpans, len(intersectSpan.spans))
	indexes := make([]*base.IndexCost, 0, len(intersectSpan.spans))
	for _, span := range intersectSpan.spans {
		skipKeys := make([]bool, len(sargKeys))
		tcost, tsel, tcard, tsize, tfrCost, e := indexScanCost(index, sargKeys, requestId, span, alias, keyspace,
			advisorValidate, context)
		if e != nil {
			return tcost, tsel, tcard, tsize, tfrCost, e
		}
		tfetchCost, _, _ := optutil.CalcFetchCost(keyspace, tcard)
		if tcost <= 0.0 || tsel <= 0.0 || tcard <= 0.0 || tfrCost <= 0.0 || tfetchCost <= 0.0 {
			return OPT_COST_NOT_AVAIL, OPT_SELEC_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL, nil
		}
		icost := base.NewIndexCost(index, tcost, tcard, tsel, tsize, tfrCost, tfetchCost,
			OPT_COST_NOT_AVAIL, OPT_COST_NOT_AVAIL, skipKeys)
		indexes = append(indexes, icost)
		spanMap[icost] = span
	}

	indexes = optutil.ChooseIntersectScan(datastore.IndexQualifiedKeyspacePath(index), indexes, -1)

	var cost, sel, frCost, nrows float64
	var size int64
	newSpans := make([]SargSpans, 0, len(indexes))
	for i, ic := range indexes {
		tcost := ic.Cost()
		tcard := ic.Cardinality()
		tsize := ic.Size()
		tfrCost := ic.FrCost()
		tsel := ic.Selectivity()
		cost += tcost
		tnrows := tcard / tsel
		if i == 0 {
			sel = tsel
			nrows = tnrows
			frCost = tfrCost
			size = tsize
		} else {
			tsel = tsel * (tnrows / nrows)
			sel = sel * tsel
			if tsize > size {
				size = tsize
			}
		}
		if span, ok := spanMap[ic]; ok {
			newSpans = append(newSpans, span)
		} else {
			return OPT_COST_NOT_AVAIL, OPT_SELEC_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL,
				errors.NewPlanInternalError("intersectSpansCost: map corrupted")
		}
	}
	intersectSpan.spans = newSpans
	return cost, sel, (sel * nrows), size, frCost, nil
}

func indexSelec(index datastore.Index, sargKeys expression.Expressions, skipKeys []bool,
	spans SargSpans, alias string, considerInternal bool, context *PrepareContext) (
	sel float64, err error) {
	switch spans := spans.(type) {
	case *TermSpans:
		sel, _ := optutil.CalcIndexSelec(index, "", sargKeys, skipKeys, spans.spans, alias, considerInternal, context)
		return sel, nil
	case *IntersectSpans:
		return multiIndexSelec(index, sargKeys, skipKeys, spans.spans, alias, false, considerInternal, context)
	case *UnionSpans:
		return multiIndexSelec(index, sargKeys, skipKeys, spans.spans, alias, true, considerInternal, context)
	}

	return OPT_SELEC_NOT_AVAIL, errors.NewPlanInternalError("indexSelec: unexpected span type")
}

func multiIndexSelec(index datastore.Index, sargKeys expression.Expressions, skipKeys []bool,
	spans []SargSpans, alias string, union, considerInternal bool, context *PrepareContext) (
	sel float64, err error) {
	for i, span := range spans {
		tsel, e := indexSelec(index, sargKeys, skipKeys, span, alias, considerInternal, context)
		if e != nil {
			return tsel, e
		}
		if i == 0 {
			sel = tsel
		} else {
			if union {
				sel = sel + tsel - (sel * tsel)
			} else {
				sel = sel * tsel
			}
		}
	}

	return sel, nil
}

func (this *builder) getIndexLimitCost(cost, cardinality, frCost, selec float64) (float64, float64, float64, float64) {
	namedArgs := this.context.NamedArgs()
	positionalArgs := this.context.PositionalArgs()

	nlimit := int64(-1)
	noffset := int64(-1)
	limit := this.limit
	offset := this.offset
	if len(namedArgs) > 0 || len(positionalArgs) > 0 {
		var err error
		limit, err = base.ReplaceParameters(limit, namedArgs, positionalArgs)
		if err != nil {
			return cost, cardinality, frCost, selec
		}
		if offset != nil {
			offset, err = base.ReplaceParameters(offset, namedArgs, positionalArgs)
			if err != nil {
				return cost, cardinality, frCost, selec
			}
		}
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

	return optutil.IndexLimitCost(nlimit, noffset, cost, cardinality, frCost, selec)
}

func getIndexProjectionCost(index datastore.Index, indexProjection *plan.IndexProjection,
	cardinality float64) (float64, float64, int64, float64) {
	return optutil.CalcIndexProjectionCost(index, indexProjection, cardinality, 0, 0, 0)
}

func getIndexGroupAggsCost(index datastore.Index, indexGroupAggs *plan.IndexGroupAggregates,
	indexProjection *plan.IndexProjection, keyspaces map[string]string,
	cardinality float64) (float64, float64, int64, float64) {
	return optutil.CalcIndexGroupAggsCost(index, indexGroupAggs, indexProjection, keyspaces, cardinality)
}

func getKeyScanCost(keys expression.Expression) (float64, float64, int64, float64) {
	return optutil.CalcKeyScanCost(keys)
}

func getFetchCost(keyspaceName string, cardinality float64) (float64, int64, float64) {
	if keyspaceName == "" {
		return OPT_COST_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL
	}
	return optutil.CalcFetchCost(keyspaceName, cardinality)
}

func getDistinctScanCost(index datastore.Index, cardinality float64, spans plan.Spans2,
	baseKeyspace *base.BaseKeyspace) (float64, float64, float64) {
	return optutil.CalcDistinctScanCost(index, cardinality, true, spans, baseKeyspace)
}

func getExpressionScanCost(expr expression.Expression) (float64, float64, int64, float64) {
	return optutil.CalcExpressionScanCost(expr)
}

func getValueScanCost(pairs algebra.Pairs) (float64, float64, int64, float64) {
	return optutil.CalcValueScanCost(pairs)
}

func getDummyScanCost() (float64, float64, int64, float64) {
	return optutil.CalcDummyScanCost()
}

func getCountScanCost() (float64, float64, int64, float64) {
	return optutil.CalcCountScanCost()
}

func getNLJoinCost(left, right plan.Operator, filters base.Filters, outer bool, op string) (
	float64, float64, int64, float64) {
	jointype := optutil.COST_JOIN
	if op == "nest" {
		jointype = optutil.COST_NEST
	}
	return optutil.CalcNLJoinCost(left, right, filters, outer, jointype)
}

func getHashJoinCost(left, right plan.Operator, buildExprs, probeExprs expression.Expressions,
	buildRight, force bool, filters base.Filters, outer bool, op string) (
	float64, float64, int64, float64, bool) {
	jointype := optutil.COST_JOIN
	if op == "nest" {
		jointype = optutil.COST_NEST
	}
	return optutil.CalcHashJoinCost(left, right, buildExprs, probeExprs, buildRight, force,
		filters, outer, jointype)
}

func getLookupJoinCost(left plan.Operator, outer bool, right *algebra.KeyspaceTerm,
	rightKeyspace string) (float64, float64, int64, float64) {
	return optutil.CalcLookupJoinNestCost(left, outer, right, rightKeyspace, optutil.COST_JOIN)
}

func getIndexJoinCost(left plan.Operator, outer bool, right *algebra.KeyspaceTerm,
	rightKeyspace string, covered bool, index datastore.Index, requestId string,
	advisorValidate bool, context *PrepareContext) (float64, float64, int64, float64) {
	return optutil.CalcIndexJoinNestCost(left, outer, right, rightKeyspace,
		covered, index, requestId, optutil.COST_JOIN, advisorValidate, context)
}

func getLookupNestCost(left plan.Operator, outer bool, right *algebra.KeyspaceTerm,
	rightKeyspace string) (float64, float64, int64, float64) {
	return optutil.CalcLookupJoinNestCost(left, outer, right, rightKeyspace, optutil.COST_NEST)
}

func getIndexNestCost(left plan.Operator, outer bool, right *algebra.KeyspaceTerm,
	rightKeyspace string, index datastore.Index, requestId string, advisorValidate bool,
	context *PrepareContext) (float64, float64, int64, float64) {
	return optutil.CalcIndexJoinNestCost(left, outer, right, rightKeyspace,
		false, index, requestId, optutil.COST_NEST, advisorValidate, context)
}

func getUnnestCost(node *algebra.Unnest, lastOp plan.Operator,
	baseKeyspaces map[string]*base.BaseKeyspace, keyspaceNames map[string]string,
	advisorValidate bool) (float64, float64, int64, float64) {
	return optutil.CalcUnnestCost(node, lastOp, baseKeyspaces, keyspaceNames, advisorValidate)
}

func getSimpleFromTermCost(left, right plan.Operator, filters base.Filters, outer bool, op string) (
	float64, float64, int64, float64) {
	jointype := optutil.COST_JOIN
	if op == "nest" {
		jointype = optutil.COST_NEST
	}
	return optutil.CalcSimpleFromTermCost(left, right, filters, outer, jointype)
}

func getSimpleFilterCost(alias string, cost, cardinality, selec float64, size int64, frCost float64) (
	float64, float64, int64, float64) {
	return optutil.CalcSimpleFilterCost(alias, cost, cardinality, selec, size, frCost)
}

func getFilterCost(lastOp plan.Operator, expr expression.Expression,
	baseKeyspaces map[string]*base.BaseKeyspace, keyspaceNames map[string]string,
	alias string, advisorValidate bool, context *PrepareContext) (float64, float64, int64, float64) {
	return optutil.CalcFilterCost(lastOp, expr, baseKeyspaces, keyspaceNames, alias, advisorValidate, context)
}

func getFilterCostWithInput(expr expression.Expression, baseKeyspaces map[string]*base.BaseKeyspace,
	keyspaceNames map[string]string, alias string, cost, cardinality float64, size int64, frCost float64,
	advisorValidate bool, context *PrepareContext) (float64, float64, int64, float64) {
	return optutil.CalcFilterCostWithInput(expr, baseKeyspaces, keyspaceNames, alias,
		cost, cardinality, size, frCost, advisorValidate, context)
}

func getLetCost(lastOp plan.Operator) (float64, float64, int64, float64) {
	return optutil.CalcLetCost(lastOp)
}

func getWithCost(lastOp plan.Operator, with expression.Withs) (float64, float64, int64, float64) {
	return optutil.CalcWithCost(lastOp, with)
}

func getOffsetCost(lastOp plan.Operator, noffset int64) (float64, float64, int64, float64) {
	return optutil.CalcOffsetCost(lastOp, noffset)
}

func getLimitCost(lastOp plan.Operator, nlimit, noffset int64) (float64, float64, int64, float64) {
	return optutil.CalcLimitCost(lastOp, nlimit, noffset)
}

func getUnnestPredSelec(pred expression.Expression, variable string, mapping expression.Expression,
	keyspaces map[string]string, advisorValidate bool, context *PrepareContext) float64 {
	return optutil.GetUnnestPredSelec(pred, variable, mapping, keyspaces, advisorValidate, context)
}

func optChooseIntersectScan(keyspace datastore.Keyspace, sargables map[datastore.Index]*indexEntry,
	nTerms int, baseKeyspace *base.BaseKeyspace, limit, offset expression.Expression,
	advisorValidate, singleKeyspace bool, context *PrepareContext) map[datastore.Index]*indexEntry {

	if keyspace == nil {
		return sargables
	}

	indexes := make([]*base.IndexCost, 0, len(sargables))

	hasPdOrder := false
	hasEarlyOrder := false
	for s, e := range sargables {
		skipKeys := make([]bool, len(e.sargKeys))
		selectivity := e.selectivity
		cardinality := e.cardinality
		if e.HasFlag(IE_ARRAYINDEXKEY_SARGABLE) {
			// array index with sargable array index key
			selec := optutil.CalcDistinctScanSelec(s, selectivity, e.arrayKeyPos, advisorValidate)
			if selec > 0.0 {
				selectivity, cardinality = optutil.AdjustArraySelec(s, selec, cardinality)
			}
		} else if e.HasFlag(IE_ARRAYINDEXKEY) {
			// array index without sargable array index key
			selectivity, cardinality = optutil.AdjustArraySelec(s, selectivity, cardinality)
		}
		icost := base.NewIndexCost(s, e.cost, cardinality, selectivity, e.size, e.frCost,
			e.fetchCost, OPT_COST_NOT_AVAIL, OPT_COST_NOT_AVAIL, skipKeys)
		if e.IsPushDownProperty(_PUSHDOWN_ORDER) {
			icost.SetPdOrder()
			hasPdOrder = true
			if e.IsPushDownProperty(_PUSHDOWN_EXACTSPANS) {
				icost.SetExactSpans()
			}
		} else if e.HasFlag(IE_HAS_EARLY_ORDER) {
			icost.SetEarlyOrder()
			if e.IsPushDownProperty(_PUSHDOWN_EXACTSPANS) {
				icost.SetExactSpans()
			}
			hasEarlyOrder = true
		}
		indexes = append(indexes, icost)
	}

	var nlimit, noffset int64
	if hasPdOrder && singleKeyspace {
		allSelec := optutil.GetAllSelec(baseKeyspace.Filters())
		if allSelec > 0.0 {
			nlimit, _ = base.GetStaticInt(limit)
			noffset, _ = base.GetStaticInt(offset)
			if nlimit > 0 && noffset > 0 {
				nlimit += noffset
			}
			if nlimit > 0 {
				for _, idx := range indexes {
					if !idx.HasPdOrder() {
						continue
					}
					// account for "limit" in pushdown order
					selec := idx.Selectivity()
					cardinality := idx.Cardinality()
					tlimit := int64(float64(nlimit)*selec/allSelec + 0.5)
					if cardinality > float64(tlimit) && tlimit >= 1 {
						limitCost, _, _, _ := optutil.CalcLimitCostInput(idx.Cost(),
							cardinality, idx.Size(), idx.FrCost(), tlimit, -1)
						fetchCost, _, fetchFrCost := optutil.CalcFetchCost(baseKeyspace.Keyspace(), cardinality)
						limitFetchCost := fetchFrCost + (fetchCost-fetchFrCost)*float64(tlimit-1)/(cardinality-1)
						idx.SetLimitCost(limitCost, limitFetchCost)
					}
				}
			}
		}
	}
	if hasPdOrder && nTerms > 0 {
		// If some plans have Order pushdown, then add a SORT cost to all plans that
		// do not have Order pushdown.
		// Note that since we are still at keyspace level, the SORT cost is not going
		// to be the same as actual SORT cost which is done at the top of the plan,
		// however this is the best estimation we could do at this level.
		// (also ignore limit and offset for this calculation).
		for _, ic := range indexes {
			if !ic.HasPdOrder() {
				sortCost, _, _, _ := getSortCost(ic.Size(), nTerms, ic.Cardinality(), nlimit, 0)
				if sortCost > 0.0 {
					ic.SetCost(ic.Cost() + sortCost)
				}
			}
		}
	}

	// pick an index scan with order if the index has _PUSHDOWN_EXACTSPANS
	if hasPdOrder && hasEarlyOrder {
		// choose the best among the indexes that provide order (pushdown or early)
		var bestIndex *base.IndexCost
		var bestCost float64
		for _, idx := range indexes {
			if (!idx.HasPdOrder() && !idx.HasEarlyOrder()) || !idx.HasExactSpans() {
				continue
			}
			cost := idx.Cost()
			if nlimit > 0 && idx.HasPdOrder() {
				// account for "limit" in pushdown order
				limitCost := idx.LimitCost()
				if limitCost > 0.0 {
					cost = limitCost
				}
			}
			if bestIndex == nil {
				bestIndex = idx
				bestCost = cost
			} else if cost < bestCost ||
				(cost == bestCost && idx.Cardinality() < bestIndex.Cardinality()) {
				bestIndex = idx
				bestCost = cost
			}
		}
		if bestIndex != nil {
			index := bestIndex.Index()
			return map[datastore.Index]*indexEntry{index: sargables[index]}
		}
	} else if hasEarlyOrder {
		// choose the best one with early order
		var bestIndex *base.IndexCost
		for _, idx := range indexes {
			if !idx.HasEarlyOrder() || !idx.HasExactSpans() {
				continue
			}
			if bestIndex == nil {
				bestIndex = idx
			} else if idx.Cardinality() < bestIndex.Cardinality() ||
				(idx.Cardinality() == bestIndex.Cardinality() && idx.Cost() < bestIndex.Cost()) {
				bestIndex = idx
			}
		}
		if bestIndex != nil {
			index := bestIndex.Index()
			return map[datastore.Index]*indexEntry{index: sargables[index]}
		}
	}

	adjustIndexSelectivity(indexes, sargables, baseKeyspace.Name(), nlimit > 0, advisorValidate, context)

	if nlimit > 0 && (hasPdOrder && !indexes[0].HasPdOrder()) {
		// if ORDER BY present and the 1st index does not provide index order, reset nlimit
		nlimit = 0
	}
	indexes = optutil.ChooseIntersectScan(keyspace.QualifiedName(), indexes, nlimit)

	newSargables := make(map[datastore.Index]*indexEntry, len(indexes))
	for _, idx := range indexes {
		newSargables[idx.Index()] = sargables[idx.Index()]
	}

	return newSargables
}

func adjustIndexSelectivity(indexes []*base.IndexCost, sargables map[datastore.Index]*indexEntry,
	alias string, doLimit, considerInternal bool, context *PrepareContext) {

	if len(indexes) <= 1 {
		return
	}

	// first sort the slice
	sort.Slice(indexes, func(i, j int) bool {
		return ((indexes[i].ScanCost(doLimit) < indexes[j].ScanCost(doLimit)) ||
			((indexes[i].ScanCost(doLimit) == indexes[j].ScanCost(doLimit)) &&
				(indexes[i].Selectivity() < indexes[j].Selectivity())) ||
			((indexes[i].ScanCost(doLimit) == indexes[j].ScanCost(doLimit)) &&
				(indexes[i].Selectivity() == indexes[j].Selectivity()) &&
				(indexes[i].Cardinality() < indexes[j].Cardinality())))
	})

	used := make(map[string]bool, len(sargables[indexes[0].Index()].sargKeys))
	for i, idx := range indexes {
		entry := sargables[idx.Index()]
		adjust := false
		for j, key := range entry.sargKeys {
			if idx.HasSkipKey(j) {
				continue
			}
			s := key.String()
			// for array index key, ignore the distinct part
			if arr, ok := key.(*expression.All); ok {
				s = arr.Array().String()
			}

			if i == 0 {
				// this is the best index
				used[s] = true
			} else {
				// check and adjust remaining indexes
				if _, ok := used[s]; ok {
					idx.SetSkipKey(j)
					adjust = true
				}
			}
		}
		if adjust {
			skipKeys := idx.SkipKeys()
			sel, e := indexSelec(idx.Index(), entry.sargKeys, skipKeys, entry.spans,
				alias, considerInternal, context)
			if e == nil {
				if entry.HasFlag(IE_ARRAYINDEXKEY_SARGABLE) &&
					(entry.arrayKeyPos >= len(skipKeys) || !skipKeys[entry.arrayKeyPos]) {
					selec := optutil.CalcDistinctScanSelec(idx.Index(), sel, entry.arrayKeyPos, considerInternal)
					if selec > 0 {
						sel = selec
					}
				}
				origSel := idx.Selectivity()
				origCard := idx.Cardinality()
				origFetchCost := idx.FetchCost()
				newCard := (origCard / origSel) * sel
				newFetchCost := (origFetchCost / origSel) * sel
				idx.SetSelectivity(sel)
				idx.SetCardinality(newCard)
				idx.SetFetchCost(newFetchCost)
			}
		}
	}

	// recurse on remaining indexes
	adjustIndexSelectivity(indexes[1:], sargables, alias, false, considerInternal, context)
}

func getSortCost(totalSize int64, nterms int, cardinality float64, limit, offset int64) (float64, float64, int64, float64) {
	return optutil.CalcSortCost(totalSize, nterms, cardinality, limit, offset)
}

func getInitialProjectCost(projection *algebra.Projection, cost, cardinality float64,
	size int64, frCost float64) (float64, float64, int64, float64) {
	return optutil.CalcInitialProjectionCost(projection, cost, cardinality, size, frCost)
}

func getGroupCosts(group *algebra.Group, aggregates algebra.Aggregates, cost, cardinality float64,
	size int64, keyspaces map[string]string, maxParallelism int) (
	float64, float64, float64, float64, float64, float64) {
	if maxParallelism <= 0 {
		maxParallelism = plan.GetMaxParallelism()
	}
	return optutil.CalcGroupCosts(group, aggregates, cost, cardinality, size, keyspaces, maxParallelism)
}

func getDistinctCost(terms algebra.ResultTerms, cost, cardinality float64, size int64, frCost float64,
	keyspaces map[string]string) (float64, float64, int64, float64) {
	return optutil.CalcDistinctCost(terms, cost, cardinality, size, frCost, keyspaces)
}

func getUnionDistinctCost(cost, cardinality float64, first, second plan.Operator, compatible bool) (float64, float64) {
	return optutil.CalcUnionDistinctCost(cost, cardinality, first, second, compatible)
}

func getUnionAllCost(first, second plan.Operator, compatible bool) (float64, float64, int64, float64) {
	return optutil.CalcSetOpCost(first, second, compatible, optutil.COST_UNION)
}

func getIntersectAllCost(first, second plan.Operator, compatible bool) (float64, float64, int64, float64) {
	return optutil.CalcSetOpCost(first, second, compatible, optutil.COST_INTERSECT)
}

func getExceptAllCost(first, second plan.Operator, compatible bool) (float64, float64, int64, float64) {
	return optutil.CalcSetOpCost(first, second, compatible, optutil.COST_EXCEPT)
}

func getInsertCost(key, value, options, limit expression.Expression, cost, cardinality float64,
	size int64, frCost float64) (float64, float64, int64, float64) {
	return optutil.CalcInsertCost(key, value, options, limit, cost, cardinality, size, frCost)
}

func getUpsertCost(key, value, options expression.Expression, cost, cardinality float64,
	size int64, frCost float64) (float64, float64, int64, float64) {
	return optutil.CalcUpsertCost(key, value, options, cost, cardinality, size, frCost)
}

func getDeleteCost(limit expression.Expression, cost, cardinality float64,
	size int64, frCost float64) (float64, float64, int64, float64) {
	return optutil.CalcDeleteCost(limit, cost, cardinality, size, frCost)
}

func getCloneCost(cost, cardinality float64, size int64, frCost float64) (
	float64, float64, int64, float64) {
	return optutil.CalcCloneCost(cost, cardinality, size, frCost)
}

func getUpdateSetCost(set *algebra.Set, cost, cardinality float64,
	size int64, frCost float64) (float64, float64, int64, float64) {
	return optutil.CalcUpdateSetCost(set, cost, cardinality, size, frCost)
}

func getUpdateUnsetCost(unset *algebra.Unset, cost, cardinality float64,
	size int64, frCost float64) (float64, float64, int64, float64) {
	return optutil.CalcUpdateUnsetCost(unset, cost, cardinality, size, frCost)
}

func getUpdateSendCost(limit expression.Expression, cost, cardinality float64,
	size int64, frCost float64) (float64, float64, int64, float64) {
	return optutil.CalcUpdateSendCost(limit, cost, cardinality, size, frCost)
}

func getWindowAggCost(aggs algebra.Aggregates, cost, cardinality float64, size int64, frCost float64) (
	float64, float64, int64, float64) {
	return optutil.CalcWindowAggCost(aggs, cost, cardinality, size, frCost)
}

func getKeyspaceSize(keyspace string) int64 {
	return optutil.GetKeyspaceSize(keyspace)
}

func optGetJoinFilterSelec(selec, cardinality float64) float64 {
	return optutil.GetJoinFilterSelec(selec, cardinality)
}

func optChooseJoinFilters(baseKeyspace *base.BaseKeyspace, index datastore.Index) (
	float64, float64, float64, expression.Expressions) {
	return optutil.ChooseJoinFilters(baseKeyspace, index)
}

func optBuildBitFilterSize(baseKeyspace *base.BaseKeyspace, exprs expression.Expressions) (size int) {

	return optutil.CalcBuildBitFilterSize(baseKeyspace, exprs)
}
