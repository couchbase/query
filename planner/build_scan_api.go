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
	atomic "github.com/couchbase/go-couchbase/platform"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

var MaxIndexApi atomic.AlignedInt64

func SetMaxIndexAPI(apiVersion int) {
	if apiVersion < datastore.INDEX_API_MIN || apiVersion > datastore.INDEX_API_MAX {
		apiVersion = datastore.INDEX_API_MIN
	}

	atomic.StoreInt64(&MaxIndexApi, int64(apiVersion))
}

func GetMaxIndexAPI() int {
	return int(atomic.LoadInt64(&MaxIndexApi))
}

func useIndex2API(index datastore.Index) bool {
	_, ok := index.(datastore.Index2)
	return ok && GetMaxIndexAPI() >= datastore.INDEX_API_2
}

func getIndexKeys(entry *indexEntry) (indexKeys datastore.IndexKeys) {
	if useIndex2API(entry.index) {
		index2, _ := entry.index.(datastore.Index2)
		indexKeys = index2.RangeKey2()
	}

	return
}

func (this *builder) buildIndexProjection(entry *indexEntry, exprs expression.Expressions, id expression.Expression,
	primary bool) *plan.IndexProjection {

	var size int
	if entry != nil {
		size = len(entry.keys)
		primary = primary || entry.index.IsPrimary()
	}

	indexProjection := plan.NewIndexProjection(size, primary || this.requirePrimaryKey)

	if !primary && id != nil {
		for _, expr := range exprs {
			if expr.DependsOn(id) {
				indexProjection.PrimaryKey = true
				break
			}
		}
	}

	if entry != nil {
		primaryKey := indexProjection.PrimaryKey
		allKeys := true

		if !entry.index.IsPrimary() {
			for keyPos, indexKey := range entry.keys {
				curKey := false
				for _, expr := range exprs {
					if expr.DependsOn(indexKey) {
						if id != nil && id.EquivalentTo(indexKey) {
							indexProjection.PrimaryKey = true
							primaryKey = true
						} else {
							indexProjection.EntryKeys = append(indexProjection.EntryKeys, keyPos)
						}
						curKey = true
						break
					}
				}
				allKeys = allKeys && curKey
			}
		}

		if allKeys && primaryKey {
			indexProjection = nil
		}
	}

	return indexProjection
}

func (this *builder) buildIndexCountScan(node *algebra.KeyspaceTerm, entry *indexEntry,
	pred expression.Expression, distinct bool, covers expression.Covers,
	filterCovers map[*expression.Cover]value.Value) plan.SecondaryScan {

	countIndex, ok := entry.index.(datastore.CountIndex)
	if !ok {
		return nil
	}

	termSpans, ok := entry.spans.(*TermSpans)
	if !ok || (termSpans.Size() > 1 && pred.MayOverlapSpans()) {
		return nil
	}

	if distinct {
		countIndex2, ok := countIndex.(datastore.CountIndex2)
		if ok && useIndex2API(entry.index) && countIndex2.CanCountDistinct() {
			this.maxParallelism = 1
			return plan.NewIndexCountDistinctScan2(countIndex2, node, termSpans.Spans(), covers, filterCovers)
		}

		return nil
	}

	if countIndex2, ok := countIndex.(datastore.CountIndex2); ok && useIndex2API(entry.index) {
		this.maxParallelism = 1
		return plan.NewIndexCountScan2(countIndex2, node, termSpans.Spans(), covers, filterCovers)
	}

	spans, exact := ConvertSpans2ToSpan(termSpans.Spans(), len(entry.index.RangeKey()))
	if exact {
		this.maxParallelism = 1
		return plan.NewIndexCountScan(countIndex, node, spans, covers, filterCovers)
	}

	return nil
}

func (this *builder) checkPushDowns(entry *indexEntry, pred expression.Expression, alias string, array bool) (bool, error) {

	if !entry.exactSpans {
		return false, nil
	}

	// check for non sargable key is in predicate
	exprs, _, err := indexCoverExpressions(entry, entry.sargKeys, pred)
	if err != nil {
		return false, err
	}

	if !pred.CoveredBy(alias, exprs) {
		return false, err
	}

	if this.offset != nil {
		if !useIndex2API(entry.index) || !entry.spans.CanPushDownOffset(entry.index, pred.MayOverlapSpans(), array) {
			this.limit = offsetPlusLimit(this.offset, this.limit)
			this.resetOffset()
		}
	}

	return true, nil
}
