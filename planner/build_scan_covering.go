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
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

func (this *builder) buildCoveringScan(indexes map[datastore.Index]*indexEntry,
	node *algebra.KeyspaceTerm, id, pred expression.Expression) (
	plan.SecondaryScan, int, error) {

	if this.cover == nil {
		return nil, 0, nil
	}

	alias := node.Alias()
	exprs := this.cover.Expressions()

	arrays := _ARRAY_POOL.Get()
	defer _ARRAY_POOL.Put(arrays)

	covering := _COVERING_POOL.Get()
	defer _COVERING_POOL.Put(covering)

	// Remember filter covers
	fc := make(map[datastore.Index]map[*expression.Cover]value.Value, len(indexes))

outer:
	for index, entry := range indexes {
		hasArrayKey := indexHasArrayIndexKey(index)
		if hasArrayKey && (len(arrays) < len(covering)) {
			continue
		}

		// Sarg to set spans
		_, err := sargIndexes(map[datastore.Index]*indexEntry{index: entry}, pred)
		if err != nil {
			return nil, 0, err
		}

		keys := entry.keys

		// Matches execution.spanScan.RunOnce()
		if !index.IsPrimary() {
			keys = append(keys, id)
		}

		// Include filter covers
		coveringExprs, filterCovers, err := indexCoverExpressions(entry, keys, pred)
		if err != nil {
			return nil, 0, err
		}

		// Skip non-covering index
		for _, expr := range exprs {
			if !expr.CoveredBy(alias, coveringExprs) {
				continue outer
			}
		}

		if hasArrayKey {
			arrays[index] = true
		}

		covering[index] = true
		fc[index] = filterCovers
	}

	// No covering index available
	if len(covering) == 0 {
		return nil, 0, nil
	}

	// Avoid array indexes if possible
	if len(arrays) < len(covering) {
		for a, _ := range arrays {
			delete(covering, a)
		}
	}

	// Keep indexes with max sumKeys
	sumKeys := 0
	for c, _ := range covering {
		if max := indexes[c].sumKeys; max > sumKeys {
			sumKeys = max
		}
	}

	for c, _ := range covering {
		if indexes[c].sumKeys < sumKeys {
			delete(covering, c)
		}
	}

	// Use shortest remaining index
	var index datastore.Index
	for c, _ := range covering {
		if index == nil {
			index = c
		} else if len(c.RangeKey()) < len(index.RangeKey()) {
			index = c
		}
	}

	entry := indexes[index]
	sargLength := len(entry.sargKeys)
	keys := entry.keys

	// Matches execution.spanScan.RunOnce()
	if !index.IsPrimary() {
		keys = append(keys, id)
	}

	// Include covering expression from index WHERE clause
	filterCovers := fc[index]

	// Include covering expression from index keys
	covers := make(expression.Covers, 0, len(keys))
	for _, key := range keys {
		covers = append(covers, expression.NewCover(key))
	}

	arrayIndex := arrays[index]
	duplicates := entry.spans.CanHaveDuplicates(index, pred.MayOverlapSpans(), false)
	indexProjection := this.buildIndexProjection(entry, exprs, id, index.IsPrimary() || arrayIndex || duplicates)
	pushDown, err := this.checkPushDowns(entry, pred, alias, false)
	if err != nil {
		return nil, 0, err
	}

	if pushDown {
		scan := this.buildCoveringPushdDownScan(index, node, entry, pred, indexProjection,
			!arrayIndex, false, covers, filterCovers)
		if scan != nil {
			return scan, sargLength, nil
		}

	}

	this.resetCountMinMax()
	if this.order != nil {
		if this.useIndexOrder(entry, keys) {
			this.maxParallelism = 1
		} else {
			this.resetOrderOffsetLimit()
		}
	}

	if this.hasOffsetOrLimit() && !pushDown {
		this.resetOffsetLimit()
	}

	projDistinct := pushDown && canPushDownProjectionDistinct(index, this.projection, keys)

	scan := entry.spans.CreateScan(index, node, false, projDistinct, false, pred.MayOverlapSpans(), false,
		this.offset, this.limit, indexProjection, covers, filterCovers)
	if scan != nil {
		this.coveringScans = append(this.coveringScans, scan)
	}
	return scan, sargLength, nil
}

func (this *builder) buildCoveringPushdDownScan(index datastore.Index, node *algebra.KeyspaceTerm, entry *indexEntry,
	pred expression.Expression, indexProjection *plan.IndexProjection, countPush, array bool,
	covers expression.Covers, filterCovers map[*expression.Cover]value.Value) plan.SecondaryScan {

	countConstantDistinctOperand := false

	if countPush && (this.countAgg != nil || this.countDistinctAgg != nil) {
		var op expression.Expression
		var distinct bool

		if this.countAgg != nil {
			op = this.countAgg.Operand()
		} else {
			op = this.countDistinctAgg.Operand()
			distinct = true
			if op != nil && op.Value() != nil {
				countConstantDistinctOperand = true
			}
		}

		if !countConstantDistinctOperand && canPushDownCount(op, entry, distinct) {
			scan := this.buildIndexCountScan(node, entry, pred, distinct, covers, filterCovers)
			if scan != nil {
				this.countScan = scan
				return scan
			}
		}
	}

	if countConstantDistinctOperand || (this.minAgg != nil && canPushDownMin(this.minAgg, entry)) ||
		(this.maxAgg != nil && canPushDownMax(this.maxAgg, entry)) {
		this.maxParallelism = 1
		limit := expression.ONE_EXPR
		scan := entry.spans.CreateScan(index, node, false, false, false, pred.MayOverlapSpans(), array, nil, limit, indexProjection, covers, filterCovers)
		if scan != nil {
			this.coveringScans = append(this.coveringScans, scan)
		}
		return scan
	}

	return nil
}

func mapFilterCovers(fc map[string]value.Value) (map[*expression.Cover]value.Value, error) {
	if len(fc) == 0 {
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

func canPushDownCount(op expression.Expression, entry *indexEntry, distinct bool) bool {
	if op == nil {
		return !distinct
	}

	val := op.Value()
	if val != nil {
		return !distinct && val.Type() > value.NULL
	}

	if len(entry.sargKeys) == 0 || !op.EquivalentTo(entry.sargKeys[0]) {
		return false
	}

	return entry.spans.SkipsLeadingNulls()
}

func canPushDownMin(minAgg *algebra.Min, entry *indexEntry) bool {
	op := minAgg.Operand()
	if op.Value() != nil {
		return true
	}

	if len(entry.sargKeys) == 0 || !op.EquivalentTo(entry.sargKeys[0]) {
		return false
	}

	indexKeys := getIndexKeys(entry)
	if indexKeyIsDescCollation(0, indexKeys) {
		return false
	}

	return entry.spans.CanUseIndexOrder() && entry.spans.SkipsLeadingNulls()
}

func canPushDownMax(maxAgg *algebra.Max, entry *indexEntry) bool {
	op := maxAgg.Operand()
	if op.Value() != nil {
		return true
	}

	if len(entry.sargKeys) == 0 || !op.EquivalentTo(entry.sargKeys[0]) {
		return false
	}

	indexKeys := getIndexKeys(entry)
	if !indexKeyIsDescCollation(0, indexKeys) {
		return false
	}

	return entry.spans.CanUseIndexOrder()
}

func canPushDownProjectionDistinct(index datastore.Index, projection *algebra.Projection, indexKeys expression.Expressions) bool {
	if projection == nil || !useIndex2API(index) {
		return false
	}
	hash := _STRING_BOOL_POOL.Get()
	defer _STRING_BOOL_POOL.Put(hash)

	for _, key := range indexKeys {
		hash[key.String()] = true
	}

	for _, expr := range projection.Expressions() {
		if expr.Value() == nil {
			if _, ok := hash[expr.String()]; !ok {
				return false
			}
		}
	}

	return true
}

func indexCoverExpressions(entry *indexEntry, keys expression.Expressions, pred expression.Expression) (
	expression.Expressions, map[*expression.Cover]value.Value, error) {

	var filterCovers map[*expression.Cover]value.Value
	exprs := keys
	if entry.cond != nil {
		var err error
		fc := _FILTER_COVERS_POOL.Get()
		defer _FILTER_COVERS_POOL.Put(fc)
		fc = entry.cond.FilterCovers(fc)
		fc = entry.origCond.FilterCovers(fc)
		filterCovers, err = mapFilterCovers(fc)
		if err != nil {
			return nil, nil, err
		}
	}

	// Allow array indexes to cover ANY predicates
	if pred != nil && entry.exactSpans && indexHasArrayIndexKey(entry.index) {
		covers, err := CoversFor(pred, keys)
		if err != nil {
			return nil, nil, err
		}

		if len(covers) > 0 {
			if len(filterCovers) == 0 {
				filterCovers = covers
			} else {
				for c, v := range covers {
					if _, ok := filterCovers[c]; !ok {
						filterCovers[c] = v
					}
				}
			}
		}
	}

	if len(filterCovers) > 0 {
		exprs = make(expression.Expressions, len(keys), len(keys)+len(filterCovers))
		copy(exprs, keys)

		for c, _ := range filterCovers {
			exprs = append(exprs, c.Covered())
		}
	}

	return exprs, filterCovers, nil
}

var _ARRAY_POOL = datastore.NewIndexBoolPool(64)
var _COVERING_POOL = datastore.NewIndexBoolPool(64)
var _FILTER_COVERS_POOL = value.NewStringValuePool(32)
var _STRING_BOOL_POOL = util.NewStringBoolPool(1024)
