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

func (this *builder) buildCoveringUnnestScan(node *algebra.KeyspaceTerm, pred, limit expression.Expression,
	indexes map[datastore.Index]*indexEntry, unnestIndexes []datastore.Index,
	arrayKeys map[datastore.Index]*expression.All, unnests []*algebra.Unnest) (
	plan.SecondaryScan, int, error) {

	order := this.order
	limitExpr := this.limit
	countAgg := this.countAgg
	minAgg := this.minAgg

	cops := make(map[datastore.Index]plan.SecondaryScan, len(unnests))

	for _, index := range unnestIndexes {
		this.order = order
		this.limit = limitExpr
		this.countAgg = countAgg
		this.minAgg = minAgg

		entry := indexes[index]
		cop, cun, err := this.buildOneCoveringUnnestScan(node, pred, limit, index, entry, arrayKeys[index], unnests)
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
	this.resetOrderLimit()
	this.resetCountMin()

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

func (this *builder) buildOneCoveringUnnestScan(node *algebra.KeyspaceTerm, pred, limit expression.Expression,
	index datastore.Index, entry *indexEntry, arrayKey *expression.All, unnests []*algebra.Unnest) (
	plan.SecondaryScan, map[*algebra.Unnest]bool, error) {

	// Statement to be covered
	if this.cover == nil {
		return nil, nil, nil
	}

	// Sarg and populate spans
	op, unnest, _, err := matchUnnest(node, pred, limit, unnests[0], index, entry, arrayKey, unnests)
	if op == nil || err != nil {
		return nil, nil, err
	}

	// Include filter covers in covering expressions
	fc := _FILTER_COVERS_POOL.Get()
	defer _FILTER_COVERS_POOL.Put(fc)

	entry = entry.Copy()
	sargKey := expression.NewIdentifier(unnest.Alias())
	entry.sargKeys = expression.Expressions{sargKey}
	min := false
	if this.minAgg != nil && canPushDownMin(this.minAgg, entry) {
		min = true
	}

	// Covering expressions from index keys
	for i, key := range entry.keys {
		if i == 0 {
			entry.keys[i] = unrollArrayKeys(key, min)
		}
	}

	// Include META().id in covering expressions
	alias := node.Alias()
	id := expression.NewField(
		expression.NewMeta(expression.NewIdentifier(alias)),
		expression.NewFieldName("id", false))

	keys := append(entry.keys, id)

	// Include filter covers from array key
	var expr expression.Expression
	bindings := coveredUnnestBindings(arrayKey, min)
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
	if !pred.MayOverlapSpans() {
		coveredUnnests = make(map[*algebra.Unnest]bool, len(unnests))
		coveredExprs = make(map[expression.Expression]bool, len(unnests))

		for _, unnest := range unnests {
			unnestExpr := unnest.Expression()
			bindingExpr, ok := bindings[unnest.As()]
			if ok && unnestExpr.EquivalentTo(bindingExpr) {
				coveredUnnests[unnest] = true
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
		if !ok && !expr.CoveredBy(alias, coveringExprs) {
			return nil, nil, nil
		}
	}

	covers := make(expression.Covers, 0, len(keys))
	for i, _ := range keys {
		covers = append(covers, expression.NewCover(coveringExprs[i]))
	}

	// Covering UNNEST index using ALL ARRAY key
	array := len(coveredUnnests) > 0

	pushDown := entry.exactSpans
	if pushDown {
		if array && this.countAgg != nil && canPushDownCount(this.countAgg, entry) {
			if countIndex, ok := index.(datastore.CountIndex); ok {
				if termSpans, ok := entry.spans.(*TermSpans); ok && (termSpans.Size() == 1 || !pred.MayOverlapSpans()) {
					this.maxParallelism = 1
					this.countScan = plan.NewIndexCountScan(countIndex, node, termSpans.Spans(), covers, filterCovers)
					return this.countScan, coveredUnnests, nil
				}
			}
		}

		if this.minAgg != nil && canPushDownMin(this.minAgg, entry) {
			this.maxParallelism = 1
			limit = expression.ONE_EXPR
			scan := entry.spans.CreateScan(index, node, false, pred.MayOverlapSpans(), true, limit, covers, filterCovers)
			if scan != nil {
				this.coveringScans = append(this.coveringScans, scan)
			}
			return scan, coveredUnnests, nil
		}
	}

	this.resetCountMin()

	if limit != nil && !pushDown {
		limit = nil
		this.limit = nil
	}

	if this.order != nil && (!array || !this.useIndexOrder(entry, entry.keys)) {
		this.resetOrderLimit()
		limit = nil
	}

	if this.order != nil {
		this.maxParallelism = 1
	}

	scan := entry.spans.CreateScan(index, node, false, pred.MayOverlapSpans(), array, limit, covers, filterCovers)
	if scan != nil {
		this.coveringScans = append(this.coveringScans, scan)
	}
	return scan, coveredUnnests, nil
}

var _EMPTY_COVERED_EXPRS = make(map[expression.Expression]bool, 0)

func unrollArrayKeys(expr expression.Expression, min bool) expression.Expression {
	for all, ok := expr.(*expression.All); ok && (min || !all.Distinct()); all, ok = expr.(*expression.All) {
		if array, ok := all.Array().(*expression.Array); ok &&
			len(array.Bindings()) == 1 && !array.Bindings()[0].Descend() {
			expr = array.ValueMapping()
		} else {
			break
		}
	}

	return expr
}

func coveredUnnestBindings(key expression.Expression, min bool) map[string]expression.Expression {
	bindings := make(map[string]expression.Expression, 8)

	for all, ok := key.(*expression.All); ok && (min || !all.Distinct()); all, ok = key.(*expression.All) {
		if array, ok := all.Array().(*expression.Array); ok &&
			len(array.Bindings()) == 1 && !array.Bindings()[0].Descend() {
			binding := array.Bindings()[0]
			bindings[binding.Variable()] = binding.Expression()
			key = array.ValueMapping()
		} else {
			break
		}
	}

	return bindings
}
