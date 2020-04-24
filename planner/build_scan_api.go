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

func useIndex2API(index datastore.Index, indexApiVersion int) bool {
	_, ok := index.(datastore.Index2)
	return ok && indexApiVersion >= datastore.INDEX_API_2
}

func useIndex3API(index datastore.Index, indexApiVersion int) bool {
	_, ok := index.(datastore.Index3)
	return ok && indexApiVersion >= datastore.INDEX_API_3
}

func useSkipIndexKeys(index datastore.Index, indexApiVersion int) bool {
	return useIndex3API(index, indexApiVersion) && (index.Type() == datastore.GSI || index.Type() == datastore.VIRTUAL)
}

func getIndexKeys(entry *indexEntry) (indexKeys datastore.IndexKeys) {
	if index2, ok := entry.index.(datastore.Index2); ok {
		indexKeys = index2.RangeKey2()
	}

	return
}

func indexHasDesc(index datastore.Index) bool {
	if index2, ok := index.(datastore.Index2); ok {
		for _, key := range index2.RangeKey2() {
			if key.Desc {
				return true
			}
		}
	}

	return false
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
							curKey = true
						}
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
		if ok && useIndex2API(entry.index, this.context.IndexApiVersion()) && countIndex2.CanCountDistinct() {
			this.maxParallelism = 1
			return plan.NewIndexCountDistinctScan2(countIndex2, node, termSpans.Spans(), covers, filterCovers)
		}

		return nil
	}

	if countIndex2, ok := countIndex.(datastore.CountIndex2); ok && useIndex2API(entry.index, this.context.IndexApiVersion()) {
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
