//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plan

import (
	"fmt"
	"math"

	"github.com/couchbaselabs/query/algebra"
	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/planner"
)

func (this *builder) selectScan(keyspace datastore.Keyspace,
	node *algebra.KeyspaceTerm) (Operator, error) {
	if this.where == nil {
		return this.selectPrimaryScan(keyspace, node)
	}

	nnf := planner.NewNNF()
	where := this.where.Copy()
	where, err := nnf.Map(where)
	if err != nil {
		return nil, err
	}

	formalizer := expression.NewFormalizer()
	formalizer.Keyspace = node.Alias()

	indexers, err := keyspace.Indexers()
	if err != nil {
		return nil, err
	}

	indexes := make([]datastore.Index, 0, len(indexers)*16)

	for _, indexer := range indexers {
		idxs, err := indexer.Indexes()
		if err != nil {
			return nil, err
		}

		indexes = append(indexes, idxs...)
	}

	unfiltered := make(map[datastore.Index]expression.Expression, len(indexes))
	filtered := make(map[datastore.Index]expression.Expression, len(indexes))

	for _, index := range indexes {
		state, er := index.State()
		if er != nil {
			return nil, er
		}

		if state != datastore.ONLINE {
			continue
		}

		rangeKey := index.RangeKey()
		if len(rangeKey) == 0 || rangeKey[0] == nil {
			// Index not rangeable
			continue
		}

		key := rangeKey[0].Copy()

		key, err = formalizer.Map(key)
		if err != nil {
			return nil, err
		}

		key, err = nnf.Map(key)
		if err != nil {
			return nil, err
		}

		if !planner.SargableFor(where, key) {
			// Index not applicable
			continue
		}

		indexCond := index.Condition()
		if indexCond == nil {
			unfiltered[index] = key
			continue
		}

		indexCond = indexCond.Copy()

		indexCond, err = formalizer.Map(indexCond)
		if err != nil {
			return nil, err
		}

		indexCond, err = nnf.Map(indexCond)
		if err != nil {
			return nil, err
		}

		if planner.SubsetOf(where, indexCond) {
			// Index condition satisfies query condition
			filtered[index] = key
			break
		}
	}

	var indexMap map[datastore.Index]expression.Expression
	if len(filtered) > 0 {
		indexMap = filtered
	} else if len(unfiltered) > 0 {
		indexMap = unfiltered
	}

	for index, key := range indexMap {
		spans := planner.SargFor(where, key)
		var scan Operator
		scan = NewIndexScan(index, node, spans, false, math.MaxInt64)
		if len(spans) > 1 {
			// Use UnionScan to de-dup multiple spans
			scan = NewUnionScan(scan)
		}

		return scan, err
	}

	return this.selectPrimaryScan(keyspace, node)
}

func (this *builder) selectPrimaryScan(keyspace datastore.Keyspace,
	node *algebra.KeyspaceTerm) (Operator, error) {
	indexers, err := keyspace.Indexers()
	if err != nil {
		return nil, err
	}

	var primary datastore.PrimaryIndex
	for _, indexer := range indexers {
		primary, err = indexer.IndexByPrimary()
		if err != nil {
			return nil, err
		}

		state, er := primary.State()
		if er != nil {
			return nil, er
		}

		if state != datastore.ONLINE {
			continue
		}

		break
	}

	if primary == nil {
		return nil, fmt.Errorf(
			"No primary index on keyspace %s. Use CREATE PRIMARY INDEX to create one.",
			keyspace.Name())
	}

	state, err := primary.State()
	if err != nil {
		return nil, err
	}

	if state != datastore.ONLINE {
		return nil, fmt.Errorf("Primary index %s not online.", primary.Name())
	}

	scan := NewPrimaryScan(primary, node)
	return scan, nil
}
