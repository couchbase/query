//  Copyright (c) 2014 Couchbase, Inc.
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
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

func (this *builder) buildCoveringScan(indexes map[datastore.Index]*indexEntry,
	node *algebra.KeyspaceTerm, id, pred, limit expression.Expression) (plan.Operator, error) {
	if this.cover == nil {
		return nil, nil
	}

	alias := node.Alias()
	exprs := this.cover.Expressions()
	arrayIndexCount := 0
	covering := _COVERING_POOL.Get()
	defer _COVERING_POOL.Put(covering)

outer:
	for index, entry := range indexes {
		keys := entry.keys

		// Matches execution.spanScan.RunOnce()
		if !index.IsPrimary() {
			keys = append(keys, id)
		}

		// Include covering expression from index WHERE clause
		coveringExprs, _, err := indexKeyExpressions(entry, keys)
		if err != nil {
			return nil, err
		}

		// Use the first available covering index
		for _, expr := range exprs {
			if !expr.CoveredBy(alias, coveringExprs) {
				continue outer
			}
		}

		if indexHasArrayIndexKey(index) {
			arrayIndexCount++
		}

		covering = append(covering, index)
	}

	// No covering index available
	if len(covering) == 0 {
		return nil, nil
	}

	// Avoid array indexes if possible
	if arrayIndexCount > 0 && arrayIndexCount < len(covering) {
		for i, c := range covering {
			if indexHasArrayIndexKey(c) {
				covering[i] = nil
			}
		}
	}

	// Use shortest covering index
	index := covering[0]
	for _, c := range covering[1:] {
		if c == nil {
			continue
		} else if index == nil {
			index = c
		} else if len(c.RangeKey()) < len(index.RangeKey()) {
			index = c
		}
	}

	entry := indexes[index]
	keys := entry.keys

	// Matches execution.spanScan.RunOnce()
	if !index.IsPrimary() {
		keys = append(keys, id)
	}

	// Only sarg a single covering index
	_, err := sargIndexes(map[datastore.Index]*indexEntry{index: entry}, pred)
	if err != nil {
		return nil, err
	}

	// Include covering expression from index WHERE clause
	_, filterCovers, err := indexKeyExpressions(entry, keys)
	if err != nil {
		return nil, err
	}

	covers := make(expression.Covers, 0, len(keys))
	for _, key := range keys {
		covers = append(covers, expression.NewCover(key))
	}

	arrayIndex := indexHasArrayIndexKey(index)

	if !arrayIndex && allowedPushDown(entry, pred) {
		if this.countAgg != nil && !pred.MayOverlapSpans() && canPushDownCount(this.countAgg, entry) {
			countIndex, ok := index.(datastore.CountIndex)
			if ok {
				this.maxParallelism = 1
				this.countScan = plan.NewIndexCountScan(countIndex, node, entry.spans, covers)
				return this.countScan, nil
			}
		}

		if this.minAgg != nil && canPushDownMin(this.minAgg, entry) {
			this.maxParallelism = 1
			limit = expression.ONE_EXPR
			scan := plan.NewIndexScan(index, node, entry.spans, false, limit, covers, filterCovers)
			this.coveringScans = append(this.coveringScans, scan)
			return scan, nil
		}
	}

	if limit != nil && (arrayIndex || !allowedPushDown(entry, pred)) {
		limit = nil
		this.limit = nil
	}

	if this.order != nil && !this.useIndexOrder(entry, keys) {
		this.resetOrderLimit()
		limit = nil
	}

	if this.order != nil {
		this.maxParallelism = 1
	}

	scan := plan.NewIndexScan(index, node, entry.spans, false, limit, covers, filterCovers)
	this.coveringScans = append(this.coveringScans, scan)

	if arrayIndex || (len(entry.spans) > 1 && (!entry.exactSpans || pred.MayOverlapSpans())) {
		// Use DistinctScan to de-dup array index scans, multiple spans
		return plan.NewDistinctScan(scan), nil
	}

	return scan, nil
}

func mapFilterCovers(fc map[string]value.Value) (map[*expression.Cover]value.Value, error) {
	if fc == nil {
		return nil, nil
	}

	rv := make(map[*expression.Cover]value.Value, len(fc))
	for s, v := range fc {
		expr, err := parser.Parse(s)
		if err != nil {
			return nil, err
		}

		c := expression.NewCover(expr)
		rv[c] = v
	}

	return rv, nil
}

func canPushDownCount(countAgg *algebra.Count, entry *indexEntry) bool {
	op := countAgg.Operand()
	if op == nil {
		return true
	}

	val := op.Value()
	if val != nil {
		return val.Type() > value.NULL
	}

	if len(entry.sargKeys) == 0 || !op.EquivalentTo(entry.sargKeys[0]) {
		return false
	}

	for _, span := range entry.spans {
		if len(span.Range.Low) == 0 {
			return false
		}

		low := span.Range.Low[0]
		if low.Type() < value.NULL || (low.Type() == value.NULL && (span.Range.Inclusion&datastore.LOW) != 0) {
			return false
		}
	}
	return true
}

func canPushDownMin(minAgg *algebra.Min, entry *indexEntry) bool {
	op := minAgg.Operand()
	if len(entry.sargKeys) == 0 || !op.EquivalentTo(entry.sargKeys[0]) {
		return false
	}

	if len(entry.spans) != 1 {
		return false
	}

	span := entry.spans[0].Range
	if len(span.Low) == 0 {
		return false
	}

	low := span.Low[0]
	return low != nil &&
		(low.Type() > value.NULL ||
			(low.Type() >= value.NULL && (span.Inclusion&datastore.LOW) == 0))
}

func indexKeyExpressions(entry *indexEntry, keys expression.Expressions) (expression.Expressions, map[*expression.Cover]value.Value, error) {
	var filterCovers map[*expression.Cover]value.Value
	exprs := keys
	if entry.cond != nil {
		var err error
		fc := entry.cond.FilterCovers(make(map[string]value.Value, 16))
		fc = entry.origCond.FilterCovers(fc)

		filterCovers, err = mapFilterCovers(fc)
		if err != nil {
			return nil, nil, err
		}

		exprs = make(expression.Expressions, len(keys), len(keys)+len(filterCovers))
		copy(exprs, keys)
		for c, _ := range filterCovers {
			exprs = append(exprs, c.Covered())
		}
	}
	return exprs, filterCovers, nil
}

var _COVERING_POOL = datastore.NewIndexPool(256)
