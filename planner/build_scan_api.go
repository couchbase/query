//  Copyright 2014-Present Couchbase, Inc.
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
	"github.com/couchbase/query/util"
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

func getFlattenKeyAttributes(fks *expression.FlattenKeys, pos int) (attr datastore.IkAttributes) {
	attr = datastore.IK_NONE
	if fks.HasDesc(pos) {
		attr |= datastore.IK_DESC
	}
	if fks.HasMissing(pos) {
		attr |= datastore.IK_MISSING
	}
	return
}

func getIndexKeys(index datastore.Index) (indexKeys datastore.IndexKeys) {
	if index2, ok := index.(datastore.Index2); ok {
		indexKeys = index2.RangeKey2()
	} else {
		for _, e := range index.RangeKey() {
			indexKeys = append(indexKeys, &datastore.IndexKey{Expr: e, Attributes: datastore.IK_NONE})
		}
	}

	flattenIndexKeys := make(datastore.IndexKeys, 0, len(indexKeys))
	for _, ik := range indexKeys {
		if all, ok := ik.Expr.(*expression.All); ok && all.Flatten() {
			fkeys := all.FlattenKeys()
			for pos, fk := range fkeys.Operands() {
				fkey := all.Copy().(*expression.All)
				fkey.SetFlattenValueMapping(fk.Copy())
				attr := getFlattenKeyAttributes(fkeys, pos)
				flattenIndexKeys = append(flattenIndexKeys, &datastore.IndexKey{fkey, attr})
			}
		} else {
			flattenIndexKeys = append(flattenIndexKeys, ik)
		}
	}

	return flattenIndexKeys
}

func indexHasDesc(index datastore.Index) bool {
	if index2, ok := index.(datastore.Index2); ok {
		for _, key := range index2.RangeKey2() {
			if all, ok := key.Expr.(*expression.All); ok && all.Flatten() {
				fks := all.FlattenKeys()
				for i := 0; i < all.FlattenSize(); i++ {
					if fks.HasDesc(i) {
						return true
					}
				}
			} else if key.HasAttribute(datastore.IK_DESC) {
				return true
			}
		}
	}

	return false
}

func indexHasLeadingKeyMissingValues(index datastore.Index, controls uint64) bool {
	if index.IsPrimary() {
		return true
	}

	if util.IsFeatureEnabled(controls, util.N1QL_INDEX_MISSING) {
		keys := getIndexKeys(index)
		return len(keys) > 0 && keys[0].HasAttribute(datastore.IK_MISSING)
	}

	return false
}

func (this *builder) buildIndexProjection(entry *indexEntry, exprs expression.Expressions, id expression.Expression,
	primary bool, idxProj map[int]bool) *plan.IndexProjection {

	var size int
	if entry != nil {
		size = len(entry.keys)
		primary = primary || entry.index.IsPrimary()
	}

	primary = primary || this.requirePrimaryKey
	if !primary && id != nil {
		for _, expr := range exprs {
			if expr.DependsOn(id) {
				primary = true
				break
			}
		}
	}

	indexProjection := plan.NewIndexProjection(size, primary)

	if entry != nil {
		primaryKey := indexProjection.PrimaryKey
		allKeys := true

		if !entry.index.IsPrimary() {
			for keyPos, indexKey := range entry.keys {
				if _, ok := idxProj[keyPos]; ok {
					indexProjection.EntryKeys = append(indexProjection.EntryKeys, keyPos)
				} else {
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
		}

		if allKeys && primaryKey {
			indexProjection = nil
		} else if !primaryKey && len(indexProjection.EntryKeys) == 0 {
			// it's possible with leading missing key index to not have anything
			// necessary from the index; avoid generating an empty index projection
			// in such cases.
			indexProjection.PrimaryKey = true
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
	if !ok || (termSpans.Size() > 1 && overlapSpans(pred)) {
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

	spans, exact := ConvertSpans2ToSpan(termSpans.Spans(), getIndexSize(entry.index))
	if exact {
		this.maxParallelism = 1
		return plan.NewIndexCountScan(countIndex, node, spans, covers, filterCovers)
	}

	return nil
}

func indexHasFlattenKeys(index datastore.Index) bool {
	for _, expr := range index.RangeKey() {
		if _, _, flatten := expr.IsArrayIndexKey(); flatten {
			return true
		}
	}
	return false
}

func getIndexSize(index datastore.Index) int {
	keys := index.RangeKey()
	size := len(keys)
	for _, k := range keys {
		if all, ok := k.(*expression.All); ok && all.Flatten() {
			size += all.FlattenSize() - 1
			return size
		}
	}
	return size
}
