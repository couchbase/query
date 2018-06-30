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
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
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
	pred expression.Expression, indexes map[datastore.Index]*indexEntry) (
	op plan.SecondaryScan, sargLength int, err error) {

	// Enumerate INNER UNNESTs
	unnests := _UNNEST_POOL.Get()
	defer _UNNEST_POOL.Put(unnests)
	unnests = collectInnerUnnests(from, unnests)
	if len(unnests) == 0 {
		return nil, 0, nil
	}

	// Enumerate primary UNNESTs
	primaryUnnests := _UNNEST_POOL.Get()
	defer _UNNEST_POOL.Put(primaryUnnests)
	primaryUnnests = collectPrimaryUnnests(from, unnests, primaryUnnests)
	if len(primaryUnnests) == 0 {
		return nil, 0, nil
	}

	// Enumerate candidate array indexes
	unnestIndexes := _INDEX_POOL.Get()
	defer _INDEX_POOL.Put(unnestIndexes)
	unnestIndexes, arrayKeys := collectUnnestIndexes(pred, indexes, unnestIndexes)
	if len(unnestIndexes) == 0 {
		return nil, 0, nil
	}

	// Add INNER UNNESTs predicates for index selection
	var andBuf [16]expression.Expression
	var andTerms []expression.Expression

	nlen := 1 + len(unnests)
	for _, unnest := range unnests {
		unnestKeyspace, ok := this.baseKeyspaces[unnest.Alias()]
		if !ok {
			return nil, 0, errors.NewPlanInternalError(fmt.Sprintf("buildUnnestScan: missing baseKeyspace %s", unnest.Alias()))
		}
		nlen += len(unnestKeyspace.filters) + len(unnestKeyspace.joinfilters)
	}
	if nlen <= len(andBuf) {
		andTerms = andBuf[0:0]
	} else {
		andTerms = make(expression.Expressions, 0, nlen)
	}

	if pred != nil {
		andTerms = append(andTerms, pred.Copy())
	}

	primaries := make(map[string]expression.Expression, len(unnests))
	primaries[node.Alias()] = expression.NewIdentifier(node.Alias())

	for _, unnest := range unnests {
		andTerms = append(andTerms, expression.NewIsNotMissing(expression.NewIdentifier(unnest.Alias())))

		for _, kexpr := range primaries {
			if unnest.Expression().DependsOn(kexpr) {
				primaries[unnest.Alias()] = expression.NewIdentifier(unnest.Alias())
				break
			}
		}

		unnestKeyspace, _ := this.baseKeyspaces[unnest.Alias()]
		// MB-25949, includes predicates on the unnested alias
		for _, fl := range unnestKeyspace.filters {
			andTerms = append(andTerms, fl.fltrExpr)
		}
		// MB-28720, includes join predicates that only refer to primary term
		// MB-30292, in case of multiple levels of unnest, include join predicates
		//           that only refers to aliases in the multiple levels of unnest
		for _, jfl := range unnestKeyspace.joinfilters {
			if jfl.singleJoinFilter(primaries) {
				andTerms = append(andTerms, jfl.fltrExpr)
			}
		}
	}

	pred = expression.NewAnd(andTerms...)
	dnf := NewDNF(pred, true, true)
	pred, err = dnf.Map(pred)
	if err != nil {
		return nil, 0, err
	}

	cop, sargLength, err := this.buildCoveringUnnestScan(node, pred, indexes, unnestIndexes, arrayKeys, unnests)
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
			op, _, n, err = this.matchUnnest(node, pred, unnest, index, indexes[index], arrayKey, unnests)
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

	entries = minimalIndexesUnnest(entries, ops)

	var scanBuf [16]plan.SecondaryScan
	var scans []plan.SecondaryScan
	if len(entries) <= len(scanBuf) {
		scans = scanBuf[0:0]
	} else {
		scans = make([]plan.SecondaryScan, 0, len(entries))
	}

	for index, _ := range entries {
		scans = append(scans, ops[index].Op)
	}

	if len(scans) == 1 {
		return scans[0], 1, nil
	} else {
		return plan.NewIntersectScan(nil, scans...), 1, nil
	}
}

type opEntry struct {
	Op  plan.SecondaryScan
	Len int
}

/*
Enumerate INNER UNNEST terms.
*/
func collectInnerUnnests(from algebra.FromTerm, buf []*algebra.Unnest) []*algebra.Unnest {
	joinTerm, ok := from.(algebra.JoinTerm)
	if !ok {
		return buf
	}

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
func collectPrimaryUnnests(from algebra.FromTerm, unnests, buf []*algebra.Unnest) []*algebra.Unnest {
	primaryAlias := expression.NewIdentifier(from.PrimaryTerm().Alias())
	for _, u := range unnests {
		// This test allows false positives, but that's ok
		if u.Expression().DependsOn(primaryAlias) {
			buf = append(buf, u)
		}
	}

	return buf
}

/*
Enumerate array indexes for UNNEST.
*/
func collectUnnestIndexes(pred expression.Expression, indexes map[datastore.Index]*indexEntry,
	unnestIndexes []datastore.Index) (
	[]datastore.Index, map[datastore.Index]*expression.All) {

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
			!SubsetOf(pred, entry.cond) {
			continue
		}

		unnestIndexes = append(unnestIndexes, index)
		arrayKeys[index] = all
	}

	return unnestIndexes, arrayKeys
}

func (this *builder) matchUnnest(node *algebra.KeyspaceTerm, pred expression.Expression, unnest *algebra.Unnest,
	index datastore.Index, entry *indexEntry, arrayKey *expression.All, unnests []*algebra.Unnest) (
	plan.SecondaryScan, *algebra.Unnest, int, error) {

	var sargKey expression.Expression

	array, ok := arrayKey.Array().(*expression.Array)
	if ok {
		if len(array.Bindings()) != 1 {
			return nil, nil, 0, nil
		}

		binding := array.Bindings()[0]
		if unnest.As() != binding.Variable() ||
			!unnest.Expression().EquivalentTo(binding.Expression()) {
			return nil, nil, 0, nil
		}

		if array.When() != nil && !SubsetOf(pred, array.When()) {
			return nil, nil, 0, nil
		}

		arrayMapping := array.ValueMapping()
		nestedArrayKey, ok := arrayMapping.(*expression.All)
		if ok {
			alias := expression.NewIdentifier(unnest.As())
			for _, u := range unnests {
				if u == unnest ||
					!u.Expression().DependsOn(alias) {
					continue
				}

				op, un, n, err := this.matchUnnest(node, pred, u, index, entry, nestedArrayKey, unnests)
				if op != nil || err != nil {
					return op, un, n + 1, err
				}
			}

			return nil, nil, 0, nil
		}

		sargKey = arrayMapping
	} else if unnest.As() == "" || !unnest.Expression().EquivalentTo(arrayKey.Array()) {
		return nil, nil, 0, nil
	} else {
		sargKey = expression.NewIdentifier(unnest.As())
	}

	formalizer := expression.NewSelfFormalizer(node.Alias(), nil)
	sargKeys := make(expression.Expressions, 0, len(index.RangeKey()))
	for i, key := range index.RangeKey() {
		if i == 0 {
			sargKeys = append(sargKeys, sargKey)
		} else {
			formalizer.SetIndexScope()
			key, err := formalizer.Map(key)
			formalizer.ClearIndexScope()
			if err != nil {
				return nil, nil, 0, nil
			}
			sargKeys = append(sargKeys, key.Copy())
		}
	}
	min, _ := SargableFor(pred, sargKeys)
	if min == 0 {
		return nil, nil, 0, nil
	}

	spans, exactSpans, err := SargFor(pred, sargKeys, min, false, node.Alias())
	if err != nil {
		return nil, nil, 0, err
	}

	entry.spans = spans
	entry.exactSpans = exactSpans
	indexProjection := this.buildIndexProjection(entry, nil, nil, true)
	scan := entry.spans.CreateScan(index, node, this.indexApiVersion, false, false, pred.MayOverlapSpans(), false,
		nil, nil, indexProjection, nil, nil, nil, nil)
	return scan, unnest, 1, nil
}

func minimalIndexesUnnest(indexes map[datastore.Index]*indexEntry,
	ops map[datastore.Index]*opEntry) map[datastore.Index]*indexEntry {
	for s, se := range indexes {
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

	return indexes
}

/*
Is se narrower or equivalent to te.
*/
func narrowerOrEquivalentUnnest(se, te *indexEntry, sop, top *opEntry) bool {
	if top.Len > sop.Len {
		return false
	}

	if te.cond != nil && (se.cond == nil || !SubsetOf(se.cond, te.cond)) {
		return false
	}

outer:
	for _, tk := range te.keys {
		for _, sk := range se.keys {
			if SubsetOf(sk, tk) || sk.DependsOn(tk) {
				continue outer
			}
		}

		return false
	}

	return len(se.keys) <= len(te.keys)
}

var _UNNEST_POOL = algebra.NewUnnestPool(8)
