//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/datastore/virtual"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
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

func useIndex6API(index datastore.Index, indexApiVersion int) bool {
	_, ok := index.(datastore.Index6)
	return ok && indexApiVersion >= datastore.INDEX_API_6
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
			for keyPos, indexKey := range entry.idxKeys {
				if _, ok := idxProj[keyPos]; ok {
					indexProjection.EntryKeys = append(indexProjection.EntryKeys, keyPos)
				} else if indexKey.HasAttribute(datastore.IK_VECTOR) {
					indexProjection.EntryKeys = append(indexProjection.EntryKeys, keyPos)
				} else {
					curKey := false
					for _, expr := range exprs {
						if expr.DependsOn(indexKey.Expr) {
							if id != nil && id.EquivalentTo(indexKey.Expr) {
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

func getIndexKeyNames(alias string, index datastore.Index, projection *plan.IndexProjection) ([]string, error) {
	var keys datastore.IndexKeys
	var err error
	if !index.IsPrimary() {
		keys = getIndexKeys(index)
	}
	indexKeyNames := make([]string, 0, len(keys)+1)

	formalizer := expression.NewSelfFormalizer(alias, nil)

	nextKey := -1
	entryPos := -1
	if projection != nil && len(projection.EntryKeys) > 0 {
		entryPos = 0
		nextKey = projection.EntryKeys[entryPos]
	}
	for i := 0; i < len(keys); i++ {
		useKey := true
		done := false
		if entryPos >= 0 {
			if i < nextKey {
				// non-projected index key
				useKey = false
			} else if i == nextKey {
				// projected index key
				entryPos++
				if entryPos >= len(projection.EntryKeys) {
					done = true
				} else {
					nextKey = projection.EntryKeys[entryPos]
				}
			} else {
				return nil, errors.NewPlanInternalError(fmt.Sprintf("getIndexKeyNames: unexpected nextKey position %d (i = %d)",
					nextKey, i))
			}
		} // else all index keys are included (useKey remains true)

		// vector index key is in the index projection but no need to include it
		if useKey && keys[i].HasAttribute(datastore.IK_VECTOR) {
			useKey = false
		}

		if useKey {
			key := keys[i].Expr
			formalizer.SetIndexScope()
			key, err = formalizer.Map(key.Copy())
			formalizer.ClearIndexScope()
			if err != nil {
				return nil, err
			}
			indexKeyNames = append(indexKeyNames, expression.NewIndexKey(key).String())
		} else {
			indexKeyNames = append(indexKeyNames, "")
		}

		if done {
			break
		}
	}
	if index.IsPrimary() || projection == nil || projection.PrimaryKey {
		id := expression.NewField(expression.NewMeta(expression.NewIdentifier(alias)),
			expression.NewFieldName("id", false))

		indexKeyNames = append(indexKeyNames, expression.NewIndexKey(id).String())
	}

	return indexKeyNames, nil
}

func (this *builder) getIndexPartitionSets(partitionKeys expression.Expressions, node *algebra.KeyspaceTerm,
	pred expression.Expression, baseKeyspace *base.BaseKeyspace) (plan.IndexPartitionSets, error) {

	keyspace, err := this.getTermKeyspace(node)
	if err != nil {
		return nil, err
	}

	// use a virtual index with the partition keys passed in as index keys, such that we can
	// try to generate index spans in order to determine whether each of the partition keys
	// has equality (EQ, IN) predicates for purpose of partition elimination

	index := virtual.NewVirtualIndex(keyspace, "partitionVirtualIndex", nil, partitionKeys, nil,
		nil, false, false, datastore.INDEX_MODE_VIRTUAL, nil)

	nkeys := len(partitionKeys)
	keys := make(datastore.IndexKeys, 0, nkeys)
	for _, k := range partitionKeys {
		keys = append(keys, &datastore.IndexKey{k, datastore.IK_NONE})
	}

	min, max, sum, skeys := SargableFor(pred, nil, index, keys, true, true, nil, this.context, this.aliases)
	if min < nkeys {
		// not all partition keys sargable
		return nil, nil
	}

	entry := newIndexEntry(index, keys, max, nil, min, max, sum, nil, nil, nil, false, skeys)

	spans, exact, err := SargFor(pred, nil, entry, keys, false, nil, max, false, false,
		baseKeyspace, this.keyspaceNames, false, this.aliases, this.context)
	if err != nil || spans == nil || spans.Size() == 0 || !exact {
		return nil, err
	}

	// TermSpans only, even in case of OR clause, it's only relevant if all keys are sargable
	// in each arm of the OR, in which case we use the entire OR clause and generate TermSpans
	// with multiple spans (instead of UnionSpans)
	var indexPartitionSets plan.IndexPartitionSets
	if tspans, ok := spans.(*TermSpans); ok {
		indexPartitionSets = make(plan.IndexPartitionSets, len(tspans.spans))
		for i, pspan := range tspans.spans {
			if len(pspan.Ranges) != nkeys {
				return nil, nil
			}
			indexPartitionSet := make(expression.Expressions, len(pspan.Ranges))
			for j, rg := range pspan.Ranges {
				if !rg.EqualRange() {
					return nil, nil
				}
				indexPartitionSet[j] = rg.Low
			}
			indexPartitionSets[i] = plan.NewIndexPartitionSet(indexPartitionSet)
		}
	}

	return indexPartitionSets, nil
}
