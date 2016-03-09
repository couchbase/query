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
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

func (this *builder) buildCoveringScan(secondaries map[datastore.Index]*indexEntry,
	node *algebra.KeyspaceTerm, pred, limit expression.Expression) (plan.Operator, error) {
	if this.cover == nil {
		return nil, nil
	}

	alias := node.Alias()
	exprs := this.cover.Expressions()
	id := expression.NewField(
		expression.NewMeta(expression.NewIdentifier(node.Alias())),
		expression.NewFieldName("id", false))

outer:
	for index, entry := range secondaries {
		keys := entry.keys
		if !index.IsPrimary() {
			// Matches execution.spanScan.RunOnce()
			keys = append(keys, id)
		}

		// Use the first available covering index
		for _, expr := range exprs {
			if !expr.CoveredBy(alias, keys) {
				continue outer
			}
		}

		covers := make(expression.Covers, 0, len(keys))
		for _, key := range keys {
			covers = append(covers, expression.NewCover(key))
		}
		if this.order != nil && !this.useIndexOrder(entry, keys) {
			this.resetOrderLimit()
			limit = nil
		}

		if limit != nil && !pred.CoveredBy(alias, keys) {
			this.limit = nil
			limit = nil
		}

		if this.order != nil {
			this.maxParallelism = 1
		}

		if pred.IsLimitPushable() && len(entry.spans) == 1 {
			countIndex, ok := index.(datastore.CountIndex)
			if ok {
				op := this.countOperand
				var val value.Value
				if op != nil {
					val = op.Value()
				}

				if op == nil || (val != nil && val.Type() > value.NULL) {
					this.countScan = plan.NewIndexCountScan(countIndex, node, entry.spans, covers)
					return this.countScan, nil
				}
			}
		}

		scan := plan.NewIndexScan(index, node, entry.spans, false, limit, covers)
		this.coveringScan = scan

		if len(entry.spans) > 1 {
			// Use UnionScan to de-dup multiple spans

			return plan.NewUnionScan(scan), nil
		}

		return scan, nil
	}

	return nil, nil
}
