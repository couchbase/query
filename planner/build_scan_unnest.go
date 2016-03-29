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
	"github.com/couchbase/query/value"
)

/*

Algorithm for exploiting array indexes with UNNEST.

Consider only INNER UNNESTs. OUTER UNNESTs cannot exploit array
indexing.

Return the first combination of UNNESTs and array index that works. No
optimization is performed to find a more optimal combination.

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
	pred expression.Expression, indexes map[datastore.Index]*indexEntry) (op plan.Operator, err error) {

	// Enumerate INNER UNNESTs
	unnests := _UNNEST_POOL.Get()
	defer _UNNEST_POOL.Put(unnests)
	unnests = collectInnerUnnests(from, unnests)
	if len(unnests) == 0 {
		return nil, nil
	}

	// Enumerate primary UNNESTs.
	primaryUnnests := _UNNEST_POOL.Get()
	defer _UNNEST_POOL.Put(primaryUnnests)
	primaryUnnests = collectPrimaryUnnests(from, unnests, primaryUnnests)
	if len(primaryUnnests) == 0 {
		return nil, nil
	}

	// Enumerate candidate array indexes
	unnestIndexes := _INDEX_POOL.Get()
	defer _INDEX_POOL.Put(unnestIndexes)
	unnestIndexes, allMap := collectUnnestIndexes(pred, indexes, unnestIndexes)
	if len(unnestIndexes) == 0 {
		return nil, nil
	}

	var un *algebra.Unnest
	for _, unnest := range primaryUnnests {
		for _, index := range unnestIndexes {
			mapping := allMap[index]
			op, un, err = matchUnnest(node, pred, unnest, index, indexes[index], mapping, unnests)
			if err != nil {
				return nil, err
			}

			if op == nil {
				continue
			}

			cop, err := this.buildUnnestCoveringScan(node, pred, index, indexes[index], un)
			if cop != nil || err != nil {
				return cop, err
			} else {
				return op, nil
			}
		}
	}

	return nil, nil
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
	unnestIndexes []datastore.Index) ([]datastore.Index, map[datastore.Index]*expression.All) {
	allMap := make(map[datastore.Index]*expression.All, len(indexes))

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
		allMap[index] = all
	}

	return unnestIndexes, allMap
}

func matchUnnest(node *algebra.KeyspaceTerm, pred expression.Expression, unnest *algebra.Unnest,
	index datastore.Index, entry *indexEntry, mapping *expression.All, unnests []*algebra.Unnest) (
	plan.Operator, *algebra.Unnest, error) {

	array, ok := mapping.Array().(*expression.Array)
	if !ok {
		return nil, nil, nil
	}

	if len(array.Bindings()) != 1 {
		return nil, nil, nil
	}

	binding := array.Bindings()[0]
	if unnest.As() != binding.Variable() ||
		!unnest.Expression().EquivalentTo(binding.Expression()) {
		return nil, nil, nil
	}

	arrayMapping := array.Mapping()
	nestedMapping, ok := arrayMapping.(*expression.All)
	if ok {
		alias := expression.NewIdentifier(unnest.As())
		for _, u := range unnests {
			if u == unnest ||
				!u.Expression().DependsOn(alias) {
				continue
			}

			op, un, err := matchUnnest(node, pred, u, index, entry, nestedMapping, unnests)
			if op != nil || err != nil {
				return op, un, err
			}
		}

		return nil, nil, nil
	} else {
		mappings := expression.Expressions{array.Mapping()}
		if SargableFor(pred, mappings) == 0 {
			return nil, nil, nil
		}

		spans, err := SargFor(pred, mappings, len(mappings))
		if err != nil {
			return nil, nil, err
		}

		entry.spans = spans
		scan := plan.NewIndexScan(index, node, spans, false, nil, nil, nil)
		return plan.NewDistinctScan(scan), unnest, nil
	}
}

func (this *builder) buildUnnestCoveringScan(node *algebra.KeyspaceTerm, pred expression.Expression,
	index datastore.Index, entry *indexEntry, unnest *algebra.Unnest) (plan.Operator, error) {
	if this.cover == nil {
		return nil, nil
	}

	alias := node.Alias()
	id := expression.NewField(
		expression.NewMeta(expression.NewIdentifier(alias)),
		expression.NewFieldName("id", false))

	keys := append(entry.keys, id)

	// Include covering expression from index WHERE clause
	coveringExprs := keys
	var filterCovers map[*expression.Cover]value.Value

	if entry.cond != nil {
		var err error
		fc := entry.cond.FilterCovers(make(map[string]value.Value, 16))
		filterCovers, err = mapFilterCovers(fc)
		if err != nil {
			return nil, err
		}

		coveringExprs = make(expression.Expressions, len(keys), len(keys)+len(filterCovers))
		copy(coveringExprs, keys)
		for c, _ := range filterCovers {
			coveringExprs = append(coveringExprs, c.Covered())
		}
	}

	exprs := this.cover.Expressions()
	for _, expr := range exprs {
		if !expr.CoveredBy(alias, coveringExprs) {
			return nil, nil
		}
	}

	covers := make(expression.Covers, 0, len(keys))
	for _, key := range keys {
		covers = append(covers, expression.NewCover(key))
	}

	this.resetOrderLimit()

	scan := plan.NewIndexScan(index, node, entry.spans, false, nil, covers, filterCovers)
	this.coveringScan = scan
	return plan.NewDistinctScan(scan), nil
}

var _UNNEST_POOL = algebra.NewUnnestPool(8)
