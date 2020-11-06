//  Copyright (c) 2016 Couchbase, Inc.
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
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
)

func (this *builder) buildCoveringUnnestScan(node *algebra.KeyspaceTerm, pred expression.Expression,
	indexes map[datastore.Index]*indexEntry, unnestIndexes []datastore.Index,
	arrayKeys map[datastore.Index]*expression.All, unnests []*algebra.Unnest, hasDeltaKeyspace bool) (
	plan.SecondaryScan, int, error) {

	// Statement to be covered
	if this.cover == nil || hasDeltaKeyspace {
		return nil, 0, nil
	}

	indexPushDowns := this.storeIndexPushDowns()
	cops := make(map[datastore.Index]plan.SecondaryScan, len(unnests))

	for _, index := range unnestIndexes {
		this.restoreIndexPushDowns(indexPushDowns, true)

		entry := indexes[index]
		cop, cun, err := this.buildOneCoveringUnnestScan(node, pred, entry, arrayKeys[index],
			unnests, hasDeltaKeyspace)
		if err != nil {
			return nil, 0, err
		}

		if cop == nil {
			continue
		}

		// The group, order, offset are exact (not a hint) if pushed then return immediately
		if len(cun) > 0 || this.group != nil || this.order != nil || this.offset != nil {
			this.coveredUnnests = cun
			return cop, len(entry.sargKeys), nil
		}

		cops[index] = cop
	}

	// Find shortest covering scan
	n := 0
	sargLength := 0
	var cop plan.SecondaryScan
	for index, c := range cops {
		if cop == nil || len(index.RangeKey()) < n {
			cop = c
			n = len(index.RangeKey())
			sargLength = len(indexes[index].sargKeys)
		}
	}

	// Return shortest covering scan
	if cop != nil {
		this.coveringScans = append(this.coveringScans, cop)
		this.coveredUnnests = nil
		this.resetIndexGroupAggs()
		return cop, sargLength, nil
	}

	return nil, 0, nil
}

func (this *builder) buildOneCoveringUnnestScan(node *algebra.KeyspaceTerm, pred expression.Expression,
	entry *indexEntry, arrKey *expression.All, unnests []*algebra.Unnest, hasDeltaKeyspace bool) (
	plan.SecondaryScan, map[*algebra.Unnest]bool, error) {

	// Sarg and populate spans
	op, unnest, arrayKey, _, err := this.matchUnnest(node, pred, unnests[0], entry, arrKey, unnests, hasDeltaKeyspace)
	if op == nil || err != nil {
		return nil, nil, err
	}

	// Include filter covers in covering expressions
	fc := _FILTER_COVERS_POOL.Get()
	defer _FILTER_COVERS_POOL.Put(fc)

	// Include META().id in covering expressions
	alias := node.Alias()
	baseKeyspace, _ := this.baseKeyspaces[alias]
	id := expression.NewField(
		expression.NewMeta(expression.NewIdentifier(alias)),
		expression.NewFieldName("id", false))

	index := entry.index
	entry = entry.Copy()
	unnestIdent := expression.NewIdentifier(unnest.Alias())
	unnestIdent.SetUnnestAlias(true)
	entry.sargKeys[0] = unnestIdent
	unAlias := unnest.As()
	entry.keys[0] = arrayKey
	indexArrayKey := entry.keys[0]

	allDistinct := false
	unnestExprInKeys := false
	var pushDownProperty PushDownProperties

	for _, key := range entry.keys {
		if key.EquivalentTo(unnest.Expression()) {
			unnestExprInKeys = true
			break
		}
	}

	if len(entry.keys) > 0 && !unnestExprInKeys {
		entry.keys[0] = unrollArrayKeys(indexArrayKey, true, unnest)
		if _, ok := entry.keys[0].(*expression.Identifier); ok {
			unAlias = ""
		}

		pushDownProperty = this.indexCoveringPushDownProperty(entry, append(entry.keys, id),
			unAlias, true, _PUSHDOWN_EXACTSPANS)
		allDistinct = isPushDownProperty(pushDownProperty, _PUSHDOWN_GROUPAGGS)

		entry.keys[0] = unrollArrayKeys(indexArrayKey, allDistinct, unnest)
		if _, ok := entry.keys[0].(*expression.Identifier); ok {
			unAlias = ""
		}
		entry.sargKeys[0] = entry.keys[0]
	}

	// Array index covers matching UNNEST expressions
	var coveredExprs map[expression.Expression]bool
	var coveredUnnests map[*algebra.Unnest]bool
	bindings, whens := coveredUnnestBindings(arrayKey, allDistinct, unnest)
	if !unnestExprInKeys {
		coveredUnnests = make(map[*algebra.Unnest]bool, len(unnests))
		coveredExprs = make(map[expression.Expression]bool, len(unnests))

		for _, uns := range unnests {
			unnestExpr := uns.Expression()
			bindingExpr, ok := bindings[uns.As()]
			if ok && unnestExpr.EquivalentTo(bindingExpr) {
				coveredUnnests[uns] = true
				coveredExprs[unnestExpr] = true
			} else {
				coveredUnnests = nil
				coveredExprs = _EMPTY_COVERED_EXPRS
				break
			}
		}
	}

	// Include filter covers from array key
	var expr expression.Expression
	for _, bexpr := range bindings {
		expr = expression.NewIsArray(bexpr)
		fc = expr.FilterCovers(fc)

		dnf := base.NewDNF(expr, true, true)
		expr, err = dnf.Map(expr)
		if err != nil {
			return nil, nil, err
		}
		fc = expr.FilterCovers(fc)
	}

	for _, wexpr := range whens {
		fc = wexpr.FilterCovers(fc)
	}

	// Include filter covers from index WHERE clause
	if entry.cond != nil {
		fc = entry.cond.FilterCovers(fc)
		fc = entry.origCond.FilterCovers(fc)
	}

	filterCovers, err := mapFilterCovers(fc, alias)
	if err != nil {
		return nil, nil, err
	}

	unnestFilters := make(expression.Expressions, 0, len(filterCovers)+1)
	for c, _ := range filterCovers {
		unnestFilters = append(unnestFilters, c.Covered())
	}

	// Allocate covering expressions
	var coveringBuf [64]expression.Expression
	var coveringExprs expression.Expressions
	keys := append(entry.keys, id)
	if len(keys)+len(filterCovers) <= len(coveringBuf) {
		coveringExprs = coveringBuf[0:0]
	} else {
		coveringExprs = make(expression.Expressions, 0, len(keys)+len(filterCovers))
	}

	// Covering expressions from index keys
	coveringExprs = append(coveringExprs, keys...)

	// Covering expressions from index WHERE clause
	coveringExprs = append(coveringExprs, unnestFilters...)

	// Is the statement covered by this index?
	exprs := this.cover.Expressions()
	for _, expr := range exprs {
		_, ok := coveredExprs[expr]
		if !ok && (!expression.IsCovered(expr, alias, coveringExprs) ||
			(len(coveredUnnests) > 0 && !expression.IsCovered(expr, unAlias, coveringExprs))) {

			return nil, nil, nil
		}
	}

	covers := make(expression.Covers, 0, len(keys))
	for i, _ := range keys {
		covers = append(covers, expression.NewCover(keys[i]))
	}

	// Covering UNNEST index using ALL ARRAY key
	array := len(coveredUnnests) > 0
	duplicates := entry.spans.CanHaveDuplicates(index, this.context.IndexApiVersion(), pred.MayOverlapSpans(), array)
	indexProjection := this.buildIndexProjection(entry, exprs, id, duplicates || array)
	unnestFilters = append(unnestFilters, expression.NewIsNotMissing(unnestIdent))
	entry.pushDownProperty = this.indexPushDownProperty(entry, keys, unnestFilters, pred, alias, true, true)

	// Check and reset pagination pushdows
	indexKeyOrders := this.checkResetPaginations(entry, keys)

	// Build old Aggregates on Index2 only
	scan := this.buildCoveringPushdDownIndexScan2(entry, node, baseKeyspace, pred, indexProjection,
		array, array, covers, filterCovers)
	if scan != nil {
		return scan, coveredUnnests, nil
	}

	// Aggregates check and reset
	var indexGroupAggs *plan.IndexGroupAggregates
	if !entry.IsPushDownProperty(_PUSHDOWN_GROUPAGGS) {
		this.resetIndexGroupAggs()
	}

	// build plan for aggregates
	indexGroupAggs, indexProjection = this.buildIndexGroupAggs(entry, keys, true, indexProjection)
	projDistinct := entry.IsPushDownProperty(_PUSHDOWN_DISTINCT)

	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL
	if this.useCBO && entry.cost > 0.0 && entry.cardinality > 0.0 {
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
	if indexGroupAggs == nil && !hasDeltaKeyspace {
		filter, cost, cardinality, err = this.getIndexFilter(index, node.Alias(), entry.spans,
			covers, filterCovers, entry.cost, entry.cardinality)
		if err != nil {
			return nil, nil, err
		}
		if this.useCBO {
			entry.cost = cost
			entry.cardinality = cardinality
		}
	}

	scan = entry.spans.CreateScan(index, node, this.context.IndexApiVersion(), false, projDistinct,
		pred.MayOverlapSpans(), array, this.offset, this.limit, indexProjection, indexKeyOrders,
		indexGroupAggs, covers, filterCovers, filter, entry.cost, entry.cardinality, hasDeltaKeyspace)
	if scan != nil {
		this.collectIndexKeyspaceNames(baseKeyspace.Keyspace())
		this.coveringScans = append(this.coveringScans, scan)
	}
	return scan, coveredUnnests, nil
}

var _EMPTY_COVERED_EXPRS = make(map[expression.Expression]bool, 0)

func unrollArrayKeys(expr expression.Expression, allDistinct bool, unnest *algebra.Unnest) expression.Expression {
	for all, ok := expr.(*expression.All); ok && (allDistinct || !all.Distinct()); all, ok = expr.(*expression.All) {
		if array, ok := all.Array().(*expression.Array); ok &&
			len(array.Bindings()) == 1 && !array.Bindings()[0].Descend() {
			expr = array.ValueMapping()
		} else {
			if !ok {
				unnestIdent := expression.NewIdentifier(unnest.As())
				unnestIdent.SetUnnestAlias(true)
				expr = unnestIdent
			}

			break
		}
	}

	return expr
}

func coveredUnnestBindings(key expression.Expression, allDistinct bool, unnest *algebra.Unnest) (map[string]expression.Expression, expression.Expressions) {
	bindings := make(map[string]expression.Expression, 8)
	whens := make(expression.Expressions, 0, 4)

	for all, ok := key.(*expression.All); ok && (allDistinct || !all.Distinct()); all, ok = key.(*expression.All) {
		if array, ok := all.Array().(*expression.Array); ok &&
			len(array.Bindings()) == 1 && !array.Bindings()[0].Descend() {
			binding := array.Bindings()[0]
			bindings[binding.Variable()] = binding.Expression()
			if array.When() != nil {
				whens = append(whens, array.When())
			}
			key = array.ValueMapping()
		} else {
			if !ok {
				bindings[unnest.As()] = all.Array()
			}

			break
		}
	}

	return bindings, whens
}
