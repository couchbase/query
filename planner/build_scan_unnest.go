//  Copyright 2016-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package planner

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
)

/*

Algorithm for exploiting array indexes with UNNEST.

Consider only INNER UNNESTs. OUTER UNNESTs cannot exploit array
indexing.

Return a combination of UNNESTs and array indexes that works.

To consider an array index, the array key must be the first key in the
array index, and is the only key exploited for UNNEST.

To find a combination of UNNESTs and array index:

Enumerate all INNER UNNESTs in the FROM clause. Identify the primary
UNNESTs, i.e. those that unnest data in the primary term of the FROM
clause.

Enumerate all array indexes on the primary term having the array key
as their first key. If the index has an index condition, i.e. a WHERE
clause, the query predicate must be a subset of the index
condition. These are the candidate array indexes.

For each primary UNNEST:

1. Find a candidate array index. The array index key must match the
UNNEST; i.e., the array index key is an ALL (DISTINCT) ARRAY
expression whose bindings match the UNNEST's expression and alias.

2. Determine if the index satisfies the current UNNEST, or if the
index should be considered for chained UNNESTs. If the index does not
have further dimensions, i.e. the ARRAY mapping IS NOT another ALL
(DISTINCT) ARRAY expression, then attempt to satisfy the query
predicate using the index. If the index has further dimensions,
i.e. the ARRAY mapping IS another ALL (DISTINCT) ARRAY expression,
then recursively attempt to chain another UNNEST for the index's next
dimension.

*/
func (this *builder) buildUnnestScan(node *algebra.KeyspaceTerm, from algebra.FromTerm,
	pred expression.Expression, indexes map[datastore.Index]*indexEntry, hasDeltaKeyspace bool) (
	op plan.SecondaryScan, sargLength int, err error) {

	if pred == nil {
		return nil, 0, nil
	}

	// Enumerate INNER UNNESTs
	joinTerm, ok := from.(algebra.JoinTerm)
	if !ok {
		return nil, 0, nil
	}

	unnests := _UNNEST_POOL.Get()
	defer _UNNEST_POOL.Put(unnests)
	unnests = collectInnerUnnestsFromJoinTerm(joinTerm, unnests)

	// Enumerate primary UNNESTs
	primaryUnnests := collectPrimaryUnnests(from, unnests)
	if nil != primaryUnnests {
		defer _UNNEST_POOL.Put(primaryUnnests)
	}
	if len(primaryUnnests) == 0 {
		return nil, 0, nil
	}

	// Enumerate candidate array indexes
	unnestIndexes, arrayKeys := collectUnnestIndexes(pred, indexes)
	if nil != unnestIndexes {
		defer _INDEX_POOL.Put(unnestIndexes)
	}
	if len(unnestIndexes) == 0 {
		return nil, 0, nil
	}

	cop, sargLength, err := this.buildCoveringUnnestScan(node, pred, indexes, unnestIndexes, arrayKeys,
		unnests, hasDeltaKeyspace)
	if cop != nil || err != nil {
		return cop, sargLength, err
	}

	// No pushdowns
	this.resetPushDowns()

	n := 0
	ops := make(map[datastore.Index]*opEntry, len(primaryUnnests))
	for _, unnest := range primaryUnnests {
		for _, index := range unnestIndexes {
			arrayKey := arrayKeys[index]
			op, _, _, n, err = this.matchUnnest(node, pred, unnest, indexes[index],
				arrayKey, unnests, hasDeltaKeyspace)
			if err != nil {
				return nil, 0, err
			}

			if op == nil {
				continue
			}

			// Keep the longest match for this index
			if entry, ok := ops[index]; ok && entry.Len >= n {
				continue
			} else {
				ops[index] = &opEntry{op, n}
			}
		}
	}

	// No UNNEST scan
	if len(ops) == 0 {
		return nil, 0, nil
	}

	// No pushdowns
	this.resetPushDowns()

	// Eliminate redundant scans
	entries := make(map[datastore.Index]*indexEntry, len(ops))
	for index, _ := range ops {
		entries[index] = indexes[index]
	}

	entries = this.minimalIndexesUnnest(entries, ops, node)

	var scanBuf [16]plan.SecondaryScan
	var scans []plan.SecondaryScan
	if len(entries) <= len(scanBuf) {
		scans = scanBuf[0:0]
	} else {
		scans = make([]plan.SecondaryScan, 0, len(entries))
	}

	sargLength = 1 // the unnest index key
	for index, _ := range entries {
		scans = append(scans, ops[index].Op)
		if ops[index].Len > sargLength {
			sargLength = ops[index].Len
		}
	}

	if len(scans) == 1 {
		return scans[0], sargLength, nil
	} else {
		cost, cardinality, size, frCost := this.intersectScanCost(node, scans...)
		return plan.NewIntersectScan(nil, cost, cardinality, size, frCost, scans...), sargLength, nil
	}
}

type opEntry struct {
	Op  plan.SecondaryScan
	Len int
}

func addUnnestPreds(baseKeyspaces map[string]*base.BaseKeyspace, primary *base.BaseKeyspace) {
	primaries := make(map[string]bool, len(baseKeyspaces))
	primaries[primary.Name()] = true
	nlen := 0

	for _, unnestKeyspace := range baseKeyspaces {
		if unnestKeyspace.IsPrimaryUnnest() {
			nlen += len(unnestKeyspace.Filters())
			nlen += len(unnestKeyspace.JoinFilters())
			primaries[unnestKeyspace.Name()] = true
		}
	}

	if nlen == 0 {
		return
	}

	newfilters := make(base.Filters, 0, nlen)

	for _, unnestKeyspace := range baseKeyspaces {
		if unnestKeyspace.IsPrimaryUnnest() {
			// MB-25949, includes predicates on the unnested alias
			for _, fl := range unnestKeyspace.Filters() {
				newfltr := fl.Copy()
				newfltr.SetUnnest()
				newfilters = append(newfilters, newfltr)
			}
			// MB-28720, includes join predicates that only refer to primary term
			// MB-30292, in case of multiple levels of unnest, include join predicates
			//           that only refers to aliases in the multiple levels of unnest
			for _, jfl := range unnestKeyspace.JoinFilters() {
				if jfl.SingleJoinFilter(primaries) {
					newfltr := jfl.Copy()
					newfltr.SetUnnest()
					newfilters = append(newfilters, newfltr)
				}
			}
		}
	}

	primary.AddFilters(newfilters)
	return
}

/*
Enumerate INNER UNNEST terms.
*/
func collectInnerUnnests(from algebra.FromTerm, buf []*algebra.Unnest) []*algebra.Unnest {
	joinTerm, ok := from.(algebra.JoinTerm)
	if !ok {
		return buf
	}
	return collectInnerUnnestsFromJoinTerm(joinTerm, buf)
}

func collectInnerUnnestsFromJoinTerm(joinTerm algebra.JoinTerm, buf []*algebra.Unnest) []*algebra.Unnest {
	buf = collectInnerUnnests(joinTerm.Left(), buf)

	unnest, ok := joinTerm.(*algebra.Unnest)
	if ok && !unnest.Outer() {
		buf = append(buf, unnest)
	}

	return buf
}

/*
Enumerate primary UNNESTs.
False positives are ok.
*/
func collectPrimaryUnnests(from algebra.FromTerm, unnests []*algebra.Unnest) []*algebra.Unnest {
	var buf []*algebra.Unnest
	primaryAlias := expression.NewIdentifier(from.PrimaryTerm().Alias())
	for _, u := range unnests {
		// This test allows false positives, but that's ok
		if u.Expression().DependsOn(primaryAlias) {
			if nil == buf {
				buf = _UNNEST_POOL.Get()
			}
			buf = append(buf, u)
		}
	}

	return buf
}

/*
Enumerate array indexes for UNNEST.
*/
func collectUnnestIndexes(pred expression.Expression, indexes map[datastore.Index]*indexEntry) (
	[]datastore.Index, map[datastore.Index]*expression.All) {
	var unnestIndexes []datastore.Index

	arrayKeys := make(map[datastore.Index]*expression.All, len(indexes))

	for index, entry := range indexes {
		if len(entry.keys) == 0 {
			continue
		}

		firstKey := entry.keys[0]
		all, ok := firstKey.(*expression.All)
		if !ok {
			continue
		}

		if entry.cond != nil &&
			!base.SubsetOf(pred, entry.cond) {
			continue
		}

		unnestIndexes = append(poolAllocIndexSlice(unnestIndexes), index)
		arrayKeys[index] = all
	}

	return unnestIndexes, arrayKeys
}

func (this *builder) matchUnnest(node *algebra.KeyspaceTerm, pred expression.Expression, unnest *algebra.Unnest,
	entry *indexEntry, arrayKey *expression.All, unnests []*algebra.Unnest, hasDeltaKeyspace bool) (
	plan.SecondaryScan, *algebra.Unnest, *expression.All, int, error) {

	var sargKey, origSargKey expression.Expression
	var err error

	newArrayKey := arrayKey
	array, ok := arrayKey.Array().(*expression.Array)
	if ok {
		if len(array.Bindings()) != 1 {
			return nil, nil, nil, 0, nil
		}

		binding := array.Bindings()[0]
		if !unnest.Expression().EquivalentTo(binding.Expression()) {
			return nil, nil, nil, 0, nil
		}

		var origBinding *expression.Binding
		when := array.When()
		arrayMapping := array.ValueMapping()
		alias := expression.NewIdentifier(unnest.As())
		alias.SetUnnestAlias(true)

		if unnest.As() != binding.Variable() {
			nakey, naok := arrayMapping.(*expression.All)
			for naok {
				a, aok := nakey.Array().(*expression.Array)
				// disallow if unnest alias is nested binding variable in the array index
				if !aok || len(a.Bindings()) != 1 || unnest.As() == a.Bindings()[0].Variable() {
					return nil, nil, nil, 0, nil
				}
				nakey, naok = a.ValueMapping().(*expression.All)
			}

			origBinding = binding
			binding = expression.NewSimpleBinding(unnest.As(), unnest.Expression())
			renamer := expression.NewRenamer(array.Bindings(), expression.Bindings{binding})
			if when != nil {
				when, err = renamer.Map(when.Copy())
				if err != nil {
					return nil, nil, nil, 0, nil
				}
			}
			arrayMapping, err = renamer.Map(arrayMapping.Copy())
			if err != nil {
				return nil, nil, nil, 0, nil
			}
		}

		if when != nil && !base.SubsetOf(pred, when) {
			return nil, nil, nil, 0, nil
		}

		nestedArrayKey, ok := arrayMapping.(*expression.All)
		if ok {
			for _, u := range unnests {
				if u == unnest ||
					!u.Expression().DependsOn(alias) {
					continue
				}

				op, un, nArrayKey, n, err := this.matchUnnest(node, pred, u, entry,
					nestedArrayKey, unnests, hasDeltaKeyspace)
				if err != nil {
					return nil, nil, nil, 0, err
				}

				if op != nil {
					newArrayKey = expression.NewAll(expression.NewArray(nArrayKey,
						expression.Bindings{binding}, when), arrayKey.Distinct())
					return op, un, newArrayKey, n + 1, err
				}
			}

			return nil, nil, nil, 0, nil
		}

		sargKey = arrayMapping
		if origBinding != nil {
			if unnest.As() != origBinding.Variable() {
				// remember the original mapping before binding variable replacement
				origSargKey = array.ValueMapping()
			}

			newArrayKey = expression.NewAll(expression.NewArray(arrayMapping,
				expression.Bindings{binding}, when), arrayKey.Distinct())
		}
	} else if unnest.As() == "" || !unnest.Expression().EquivalentTo(arrayKey.Array()) {
		return nil, nil, nil, 0, nil
	} else {
		unnestIdent := expression.NewIdentifier(unnest.As())
		unnestIdent.SetUnnestAlias(true)
		sargKey = unnestIdent
	}

	useCBO := this.useCBO && this.keyspaceUseCBO(node.Alias())
	advisorValidate := this.advisorValidate()
	baseKeyspace, _ := this.baseKeyspaces[node.Alias()]
	if useCBO {
		keyspaces := make(map[string]string, 1)
		keyspaces[node.Alias()] = node.Keyspace()
		for _, fl := range baseKeyspace.Filters() {
			if fl.IsUnnest() {
				sel := getUnnestPredSelec(fl.FltrExpr(), unnest.As(),
					unnest.Expression(), keyspaces, advisorValidate, this.context)
				fl.SetSelec(sel)
			}
		}
	}

	index := entry.index
	formalizer := expression.NewSelfFormalizer(node.Alias(), nil)
	sargKeys := make(expression.Expressions, 0, len(index.RangeKey()))
	for i, key := range index.RangeKey() {
		if i == 0 {
			sargKeys = append(sargKeys, sargKey)
		} else {
			formalizer.SetIndexScope()
			key, err := formalizer.Map(key.Copy())
			formalizer.ClearIndexScope()
			if err != nil {
				return nil, nil, nil, 0, nil
			}
			sargKeys = append(sargKeys, key)
		}
	}

	min, max, _, skeys := SargableFor(pred, sargKeys, false, true)
	if min == 0 {
		return nil, nil, nil, 0, nil
	}

	n := min
	if useSkipIndexKeys(index, this.context.IndexApiVersion()) {
		n = max
	}

	spans, exactSpans, err := SargFor(pred, entry, sargKeys, n, false, useCBO,
		baseKeyspace, this.keyspaceNames, advisorValidate, this.context)
	if err != nil {
		return nil, nil, nil, 0, err
	}

	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL
	selectivity := OPT_SELEC_NOT_AVAIL
	size := OPT_SIZE_NOT_AVAIL
	frCost := OPT_COST_NOT_AVAIL
	if useCBO {
		restore := false
		if origSargKey != nil {
			// use the original binding ariable in array index key
			sargKeys[0] = origSargKey
			restore = true
		}
		cost, selectivity, cardinality, size, frCost, err = indexScanCost(entry.index, sargKeys, this.context.RequestId(),
			spans, node.Alias(), this.advisorValidate(), this.context)
		if err != nil {
			cost = OPT_COST_NOT_AVAIL
			cardinality = OPT_CARD_NOT_AVAIL
			size = OPT_SIZE_NOT_AVAIL
			frCost = OPT_COST_NOT_AVAIL
		}
		if restore {
			sargKeys[0] = sargKey
		}

		baseKeyspace.AddUnnestIndex(index, unnest.Alias())
	}

	entry.sargKeys = sargKeys[0:n]
	entry.skeys = skeys
	entry.spans = spans
	entry.exactSpans = exactSpans
	entry.cost = cost
	entry.cardinality = cardinality
	entry.selectivity = selectivity
	entry.size = size
	entry.frCost = frCost
	indexProjection := this.buildIndexProjection(entry, nil, nil, true)
	this.collectIndexKeyspaceNames(baseKeyspace.Keyspace())
	scan := entry.spans.CreateScan(index, node, this.context.IndexApiVersion(), false, false,
		pred.MayOverlapSpans(), false, nil, nil, indexProjection, nil, nil, nil, nil, nil,
		cost, cardinality, size, frCost, hasDeltaKeyspace)
	return scan, unnest, newArrayKey, n, nil
}

func (this *builder) minimalIndexesUnnest(indexes map[datastore.Index]*indexEntry,
	ops map[datastore.Index]*opEntry, node *algebra.KeyspaceTerm) map[datastore.Index]*indexEntry {
	useCBO := this.useCBO && this.keyspaceUseCBO(node.Alias())
	for s, se := range indexes {
		if useCBO && (se.cost <= 0.0 || se.cardinality <= 0.0) {
			useCBO = false
		}

		for t, te := range indexes {
			if t == s {
				continue
			}

			if narrowerOrEquivalentUnnest(se, te, ops[s], ops[t]) {
				delete(indexes, t)
				delete(ops, t)
			}
		}
	}

	if useCBO && len(indexes) > 0 {
		indexes = this.chooseIntersectScan(indexes, node)
		for t, _ := range ops {
			if _, ok := indexes[t]; !ok {
				delete(ops, t)
			}
		}
	}

	return indexes
}

/*
Is se narrower or equivalent to te.
*/
func narrowerOrEquivalentUnnest(se, te *indexEntry, sop, top *opEntry) bool {
	if top.Len > sop.Len {
		return false
	}

	if te.cond != nil && (se.cond == nil || !base.SubsetOf(se.cond, te.cond)) {
		return false
	}

outer:
	for _, tk := range te.keys {
		for _, sk := range se.keys {
			if base.SubsetOf(sk, tk) || sk.DependsOn(tk) {
				continue outer
			}
		}

		return false
	}

	return len(se.keys) <= len(te.keys)
}

/*
 * collect Unnest Bindings that depends on expression
 *     recursively go through dependent expression
 *     When detects OUTER JOIN it stops
 */

func (this *builder) collectUnnestBindings(from algebra.FromTerm, ua expression.Expressions,
	ub expression.Bindings) (expression.Expressions, expression.Bindings) {

	if joinTerm, ok := from.(algebra.JoinTerm); ok {
		ua, ub = this.collectUnnestBindings(joinTerm.Left(), ua, ub)
		if unnest, ok := joinTerm.(*algebra.Unnest); ok && !unnest.Outer() {
			for _, a := range ua {
				if unnest.Expression().DependsOn(a) {
					ua = append(ua, expression.NewIdentifier(unnest.Alias()))
					ub = append(ub, expression.NewSimpleBinding(unnest.Alias(), unnest.Expression()))
					return ua, ub
				}
			}
		}
	}

	return ua, ub
}

var _UNNEST_POOL = algebra.NewUnnestPool(8)
