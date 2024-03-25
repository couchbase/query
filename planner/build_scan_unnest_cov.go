//  Copyright 2016-Present Couchbase, Inc.
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
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/value"
)

func (this *builder) buildCoveringUnnestScan(node *algebra.KeyspaceTerm,
	pred, subset, origPred, id expression.Expression, unnestIndexes map[datastore.Index]*indexEntry,
	unnests []*algebra.Unnest) (
	scan plan.SecondaryScan, sargLength int, err error) {

	// Statement to be covered
	if this.cover == nil {
		return
	}

	baseKeyspace, _ := this.baseKeyspaces[node.Alias()]
	coveringEntries := _COVERING_ENTRY_POOL.Get()
	defer _COVERING_ENTRY_POOL.Put(coveringEntries)

	for index, entry := range unnestIndexes {
		rv := &coveringEntry{idxEntry: entry, rootUnnest: unnests[0]}
		rv.idxEntry, err = this.buildOneCoveringUnnestScan(node, pred, subset, origPred, id,
			rv, rv.idxEntry.arrayKey, unnests)
		if err != nil {
			return nil, 0, err
		}

		if rv.idxEntry != nil {
			coveringEntries[index] = rv
		}
	}

	if len(coveringEntries) == 0 {
		return
	}

	index := this.bestCoveringIndex(this.useCBO, node, coveringEntries, false)
	centry := coveringEntries[index]
	implicitCover := len(centry.coveredUnnests) > 0

	var exprs expression.Expressions
	exprs, err = this.getExprsToCover()
	if err != nil {
		return
	}

	scan, sargLength, err = this.buildCreateCoveringScan(centry.idxEntry, node, id, pred,
		exprs, append(centry.idxEntry.keys, id), implicitCover,
		implicitCover, false, centry.covers, centry.filterCovers, nil)
	if err != nil || scan == nil {
		return
	}

	this.coveredUnnests = centry.coveredUnnests
	for _, a := range centry.idxEntry.unnestAliases {
		baseKeyspace.AddUnnestIndex(index, a)
	}
	return
}

func (this *builder) buildOneCoveringUnnestScan(node *algebra.KeyspaceTerm,
	pred, subset, origPred, id expression.Expression, centry *coveringEntry,
	indexArrayKey *expression.All, unnests []*algebra.Unnest) (entry *indexEntry, err error) {

	// Sarg and populate spans
	centry.idxEntry, centry.leafUnnest, indexArrayKey, err = this.matchUnnestScan(node, pred, subset,
		origPred, centry.rootUnnest, centry.idxEntry, indexArrayKey, unnests, false)
	if err != nil || centry.idxEntry == nil || centry.leafUnnest == nil || indexArrayKey == nil {
		return nil, err
	}
	entry = centry.idxEntry
	unnestExprInKeys := IsUnnestExprInIndexKeys(entry, centry.rootUnnest)
	if !unnestExprInKeys && hasUnknownsInSargableArrayKey(centry.idxEntry) {
		return nil, nil
	}
	exact := isPushDownProperty(entry.pushDownProperty, _PUSHDOWN_EXACTSPANS)
	if !unnestExprInKeys && indexArrayKey != nil && indexArrayKey.HasDescend() {
		exact = false
	}

	if entry.exactSpans && unnestExprInKeys && !exact {
		entry.exactSpans = exact
	}

	coverAliases := getUnnestAliases(entry.arrayKey, centry.leafUnnest)
	pushDownProperty := this.indexCoveringPushDownProperty(entry, append(entry.keys, id),
		node.Alias(), coverAliases, true, false, _PUSHDOWN_EXACTSPANS)
	allDistinct := isPushDownProperty(pushDownProperty, _PUSHDOWN_GROUPAGGS)
	unnestFilters, coveredExprs, filterCovers, coveredUnnests, err := this.coveringExpressions(node,
		centry.idxEntry, centry.leafUnnest, unnests, allDistinct)
	keys := make(expression.Expressions, 0, len(entry.keys)+1)
	if unnestExprInKeys {
		coveredExprs = nil
		coveredUnnests = nil
		akeys := expression.Expressions{entry.arrayKey}
		akeys = expression.CopyExpressions(expression.GetFlattenKeys(akeys))
		keys = append(keys, akeys...)
		keys = append(keys, entry.keys[len(akeys):]...)
		keys = append(keys, id)
	} else {
		keys = append(entry.keys, id)
	}
	coveringExprs := make(expression.Expressions, 0, len(keys)+len(unnestFilters))
	coveringExprs = append(coveringExprs, keys...)
	coveringExprs = append(coveringExprs, unnestFilters...)

	// Is the statement covered by this index?
	exprs, err := this.getExprsToCover()
	if err != nil {
		return nil, err
	}

	for _, expr := range exprs {
		// skip unnest expressions.Use unnest expression pointer for exact expr match.
		if _, ok := coveredExprs[expr]; ok {
			continue
		}

		if !expression.IsCovered(expr, node.Alias(), coveringExprs, false) {
			return nil, nil
		}
		if len(coveredUnnests) > 0 {
			for _, a := range coverAliases {
				if !expression.IsCovered(expr, a, coveringExprs, false) {
					return nil, nil
				}
			}
		}
	}

	covers := make(expression.Covers, 0, len(keys))
	for i, _ := range keys {
		covers = append(covers, expression.NewCover(keys[i]))
	}
	centry.covers = covers
	centry.filterCovers = filterCovers
	centry.coveredUnnests = coveredUnnests

	allKeyspaces := !unnestExprInKeys && (len(this.baseKeyspaces) == len(entry.unnestAliases)+1)

	entry.pushDownProperty = this.indexPushDownProperty(entry, keys, unnestFilters,
		pred, origPred, node.Alias(), coverAliases, true, true, allKeyspaces, false)
	if len(coveredUnnests) > 0 {
		entry.pushDownProperty |= _PUSHDOWN_COVERED_UNNEST
	} else {
		entry.SetPushDownProperty(_PUSHDOWN_NONE) // reset
	}
	return entry, nil
}

func coveredUnnestBindings(key expression.Expression, allDistinct bool,
	unnest *algebra.Unnest) (map[string]expression.Expression, expression.Expressions) {

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

func (this *builder) coveringExpressions(node *algebra.KeyspaceTerm, entry *indexEntry,
	unnest *algebra.Unnest, unnests []*algebra.Unnest, allDistinct bool) (
	unnestFilters expression.Expressions, coveredExprs map[expression.Expression]bool,
	filterCovers map[*expression.Cover]value.Value, coveredUnnests map[*algebra.Unnest]bool, err error) {

	coveredUnnests = make(map[*algebra.Unnest]bool, len(unnests))
	coveredExprs = make(map[expression.Expression]bool, len(unnests))
	unnestFilters = make(expression.Expressions, 0, 4)

	bindings, whens := coveredUnnestBindings(entry.arrayKey, allDistinct, unnest)

	for _, uns := range unnests {
		unnestExpr := uns.Expression()
		bindingExpr, ok := bindings[uns.As()]
		if ok && unnestExpr.EquivalentTo(bindingExpr) {
			coveredUnnests[uns] = true
			coveredExprs[unnestExpr] = true
		} else {
			coveredUnnests = nil
			coveredExprs = nil
			break
		}
	}

	// Include filter covers from array key
	var expr expression.Expression
	fc := make(map[expression.Expression]value.Value, len(bindings)+len(whens)+2)
	for _, bexpr := range bindings {
		expr = expression.NewIsArray(bexpr)
		fc = expr.FilterExpressionCovers(fc)

		dnf := base.NewDNF(expr, true, true)
		expr, err = dnf.Map(expr)
		if err != nil {
			return
		}
		fc = expr.FilterExpressionCovers(fc)
	}

	for _, wexpr := range whens {
		fc = wexpr.FilterExpressionCovers(fc)
	}

	// Include filter covers from index WHERE clause
	if entry.cond != nil {
		fc = entry.cond.FilterExpressionCovers(fc)
		fc = entry.origCond.FilterExpressionCovers(fc)
	}

	filterCovers = mapFilterCovers(fc, true)

	for c, _ := range filterCovers {
		unnestFilters = append(unnestFilters, c.Covered())
	}

	unnestFilters = append(unnestFilters, getUnnestFilters(entry.unnestAliases)...)

	return
}

func IsUnnestExprInIndexKeys(entry *indexEntry, unnest *algebra.Unnest) bool {
	for _, key := range entry.keys {
		if key.EquivalentTo(unnest.Expression()) {
			return true
		}
	}
	return false
}
