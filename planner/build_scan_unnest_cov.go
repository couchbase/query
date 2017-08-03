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
)

func (this *builder) buildCoveringUnnestScan(node *algebra.KeyspaceTerm, pred expression.Expression,
	indexes map[datastore.Index]*indexEntry, unnestIndexes []datastore.Index,
	arrayKeys map[datastore.Index]*expression.All, unnests []*algebra.Unnest) (
	plan.SecondaryScan, int, error) {

	order := this.order
	limitExpr := this.limit
	offsetExpr := this.offset
	countAgg := this.countAgg
	countDistinctAgg := this.countDistinctAgg
	minAgg := this.minAgg
	maxAgg := this.maxAgg

	cops := make(map[datastore.Index]plan.SecondaryScan, len(unnests))

	for _, index := range unnestIndexes {
		this.order = order
		this.limit = limitExpr
		this.offset = offsetExpr
		this.countAgg = countAgg
		this.countDistinctAgg = countDistinctAgg
		this.minAgg = minAgg
		this.maxAgg = maxAgg

		entry := indexes[index]
		cop, cun, err := this.buildOneCoveringUnnestScan(node, pred, index, entry, arrayKeys[index], unnests)
		if err != nil {
			return nil, 0, err
		}

		if cop == nil {
			continue
		}

		if len(cun) > 0 || this.minAgg != nil {
			this.coveredUnnests = cun
			return cop, len(entry.sargKeys), nil
		}

		cops[index] = cop
	}

	// No pushdowns
	this.resetPushDowns()

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
		return cop, sargLength, nil
	}

	return nil, 0, nil
}

func (this *builder) buildOneCoveringUnnestScan(node *algebra.KeyspaceTerm, pred expression.Expression,
	index datastore.Index, entry *indexEntry, arrayKey *expression.All, unnests []*algebra.Unnest) (
	plan.SecondaryScan, map[*algebra.Unnest]bool, error) {

	// Statement to be covered
	if this.cover == nil {
		return nil, nil, nil
	}

	// Sarg and populate spans
	op, unnest, _, err := this.matchUnnest(node, pred, unnests[0], index, entry, arrayKey, unnests)
	if op == nil || err != nil {
		return nil, nil, err
	}

	// Include filter covers in covering expressions
	fc := _FILTER_COVERS_POOL.Get()
	defer _FILTER_COVERS_POOL.Put(fc)

	entry = entry.Copy()
	sargKey := expression.NewIdentifier(unnest.Alias())
	entry.sargKeys = expression.Expressions{sargKey}
	allDistinct := false
	unnestExprInKeys := false
	for _, key := range entry.keys {
		if key.EquivalentTo(unnest.Expression()) {
			unnestExprInKeys = true
			break
		}
	}

	if len(entry.keys) > 0 && !unnestExprInKeys {
		unrollKeys := expression.Expressions{unrollArrayKeys(entry.keys[0], true, unnest)}
		if (this.minAgg != nil && canPushDownMin(this.minAgg, entry, unrollKeys)) ||
			(this.maxAgg != nil && canPushDownMax(this.maxAgg, entry, unrollKeys)) ||
			this.countDistinctAgg != nil {
			allDistinct = true
		}
		entry.keys[0] = unrollArrayKeys(entry.keys[0], allDistinct, unnest)
	}

	// Include META().id in covering expressions
	alias := node.Alias()
	id := expression.NewField(
		expression.NewMeta(expression.NewIdentifier(alias)),
		expression.NewFieldName("id", false))

	keys := append(entry.keys, id)

	// Include filter covers from array key
	var expr expression.Expression
	bindings := coveredUnnestBindings(arrayKey, allDistinct, unnest)
	for _, bexpr := range bindings {
		expr = expression.NewIsArray(bexpr)
		fc = expr.FilterCovers(fc)

		dnf := NewDNF(expr, true)
		expr, err = dnf.Map(expr)
		if err != nil {
			return nil, nil, err
		}
		fc = expr.FilterCovers(fc)
	}

	// Include filter covers from index WHERE clause
	if entry.cond != nil {
		fc = entry.cond.FilterCovers(fc)
		fc = entry.origCond.FilterCovers(fc)
	}

	var coveredExprs map[expression.Expression]bool
	var coveredUnnests map[*algebra.Unnest]bool

	// Array index covers matching UNNEST expressions
	if !pred.MayOverlapSpans() && !unnestExprInKeys {
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

	filterCovers, err := mapFilterCovers(fc)
	if err != nil {
		return nil, nil, err
	}

	// Allocate covering expressions
	var coveringBuf [64]expression.Expression
	var coveringExprs expression.Expressions
	if len(keys)+len(filterCovers) <= len(coveringBuf) {
		coveringExprs = coveringBuf[0:0]
	} else {
		coveringExprs = make(expression.Expressions, 0, len(keys)+len(filterCovers))
	}

	// Covering expressions from index keys
	for _, key := range keys {
		coveringExprs = append(coveringExprs, key)
	}

	// Covering expressions from index WHERE clause
	for c, _ := range filterCovers {
		coveringExprs = append(coveringExprs, c.Covered())
	}

	// Is the statement covered by this index?
	exprs := this.cover.Expressions()
	for _, expr := range exprs {
		_, ok := coveredExprs[expr]
		if !ok && (!expression.IsCovered(expr, alias, coveringExprs) ||
			(len(coveredUnnests) > 0 && !expression.IsCovered(expr, unnest.As(), coveringExprs))) {
			return nil, nil, nil
		}
	}

	covers := make(expression.Covers, 0, len(keys))
	for i, _ := range keys {
		covers = append(covers, expression.NewCover(coveringExprs[i]))
	}

	// Covering UNNEST index using ALL ARRAY key
	array := len(coveredUnnests) > 0
	duplicates := entry.spans.CanHaveDuplicates(index, pred.MayOverlapSpans(), array)
	indexProjection := this.buildIndexProjection(entry, exprs, id, duplicates || array)
	pushDown := entry.exactSpans
	if pushDown {
		scan := this.buildCoveringPushdDownScan(index, node, entry, keys[0:len(entry.sargKeys)], pred, indexProjection,
			array, array, covers, filterCovers)
		if scan != nil {
			return scan, coveredUnnests, nil
		}
	}

	this.resetCountMinMax()

	if this.order != nil {
		if array && this.useIndexOrder(entry, keys) {
			this.maxParallelism = 1
		} else {
			this.resetOrderOffsetLimit()
		}
	}

	if this.hasOffsetOrLimit() && (!array || !pushDown) {
		this.resetOffsetLimit()
	}

	projDistinct := pushDown && canPushDownProjectionDistinct(index, this.projection, keys)

	scan := entry.spans.CreateScan(index, node, false, projDistinct, false, pred.MayOverlapSpans(), array, this.offset, this.limit, indexProjection, covers, filterCovers)
	if scan != nil {
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
				expr = expression.NewIdentifier(unnest.As())
			}

			break
		}
	}

	return expr
}

func coveredUnnestBindings(key expression.Expression, allDistinct bool, unnest *algebra.Unnest) map[string]expression.Expression {
	bindings := make(map[string]expression.Expression, 8)

	for all, ok := key.(*expression.All); ok && (allDistinct || !all.Distinct()); all, ok = key.(*expression.All) {
		if array, ok := all.Array().(*expression.Array); ok &&
			len(array.Bindings()) == 1 && !array.Bindings()[0].Descend() {
			binding := array.Bindings()[0]
			bindings[binding.Variable()] = binding.Expression()
			key = array.ValueMapping()
		} else {
			if !ok {
				bindings[unnest.As()] = all.Array()
			}

			break
		}
	}

	return bindings
}
