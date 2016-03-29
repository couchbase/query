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

func (this *builder) buildCoveringScan(secondaries map[datastore.Index]*indexEntry,
	node *algebra.KeyspaceTerm, pred, limit expression.Expression) (plan.Operator, error) {
	if this.cover == nil {
		return nil, nil
	}

	alias := node.Alias()
	id := expression.NewField(
		expression.NewMeta(expression.NewIdentifier(alias)),
		expression.NewFieldName("id", false))
	exprs := this.cover.Expressions()

outer:
	for index, entry := range secondaries {
		keys := entry.keys

		// Matches execution.spanScan.RunOnce()
		if !index.IsPrimary() {
			keys = append(keys, id)
		}

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

		// Use the first available covering index
		for _, expr := range exprs {
			if !expr.CoveredBy(alias, coveringExprs) {
				continue outer
			}
		}

		covers := make(expression.Covers, 0, len(keys))
		for _, key := range keys {
			covers = append(covers, expression.NewCover(key))
		}

		arrayIndex := indexHasArrayIndexKey(index)

		if this.countAgg != nil && !arrayIndex && (len(entry.spans) == 1) && allowedPushDown(entry) {
			countIndex, ok := index.(datastore.CountIndex)
			if ok {
				op := this.countAgg.Operand()
				var val value.Value
				if op != nil {
					val = op.Value()
				}

				if op == nil || (val != nil && val.Type() > value.NULL) {
					this.maxParallelism = 1
					this.countScan = plan.NewIndexCountScan(countIndex, node, entry.spans, covers)
					return this.countScan, nil
				}
			}
		}

		if limit != nil && (arrayIndex || !allowedPushDown(entry)) {
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
		this.coveringScan = scan

		if len(entry.spans) > 1 || arrayIndex {
			// Use DistinctScan to de-dup array index scans, multiple spans
			return plan.NewDistinctScan(scan), nil
		}

		return scan, nil
	}

	return nil, nil
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
