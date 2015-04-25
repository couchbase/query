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

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/planner"
)

func (this *builder) selectScan(keyspace datastore.Keyspace,
	node *algebra.KeyspaceTerm) (op Operator, err error) {
	indexers, err := keyspace.Indexers()
	if err != nil {
		return nil, err
	}

	indexes := make([]datastore.Index, 0, len(indexers)*16)
	primaryIndexes := make(map[datastore.PrimaryIndex]bool, len(indexers)*2)

	// Hints from USE INDEX clause
	hints := node.Indexes()
	if hints != nil {
		for _, hint := range hints {
			indexer, err := keyspace.Indexer(hint.Using())
			if err != nil {
				return nil, err
			}

			index, err := indexer.IndexByName(hint.Name())
			if err != nil {
				return nil, err
			}
			state, _, er := index.State()
			if er != nil {
				logging.Errorp("Scan Selection", logging.Pair{"error", er.Error()})
			}
			if state != datastore.ONLINE {
				continue
			}

			indexes = append(indexes, index)

			if index.IsPrimary() {
				primary := index.(datastore.PrimaryIndex)
				primaryIndexes[primary] = true
			}
		}
	}

	if this.where == nil {
		return this.selectPrimaryScan(keyspace, node, primaryIndexes)
	}

	nnf := planner.NewNNF()
	where := this.where.Copy()
	where, err = nnf.Map(where)
	if err != nil {
		return nil, err
	}

	formalizer := expression.NewFormalizer()
	formalizer.Keyspace = node.Alias()
	primaryKey := expression.Expressions{
		expression.NewField(
			expression.NewMeta(expression.NewConstant(node.Alias())),
			expression.NewFieldName("id")),
	}

	if hints == nil {
		for _, indexer := range indexers {
			idxs, err := indexer.Indexes()
			if err != nil {
				return nil, err
			}

			indexes = append(indexes, idxs...)

			primaryIdxs, err := indexer.PrimaryIndexes()
			if err != nil {
				return nil, err
			}

			for _, p := range primaryIdxs {
				primaryIndexes[p] = true
			}
		}
	}

	equivalent := make(map[datastore.Index]expression.Expressions, 1)
	filtered := make(map[datastore.Index]expression.Expressions, len(indexes))
	unfiltered := make(map[datastore.Index]expression.Expressions, len(indexes))

	for _, index := range indexes {
		state, _, er := index.State()
		if er != nil {
			logging.Errorp("Scan Selection", logging.Pair{"error", er.Error()})
		}
		if state != datastore.ONLINE {
			continue
		}

		var keys expression.Expressions

		if index.IsPrimary() {
			keys = primaryKey
		} else {
			rangeKey := index.RangeKey()
			if len(rangeKey) == 0 || rangeKey[0] == nil {
				// Index not rangeable
				continue
			}

			keys = make(expression.Expressions, 0, len(rangeKey))
			for _, key := range rangeKey {
				if key == nil {
					break
				}

				key = key.Copy()

				key, err = formalizer.Map(key)
				if err != nil {
					return nil, err
				}

				key, err = nnf.Map(key)
				if err != nil {
					return nil, err
				}

				keys = append(keys, key)
			}
		}

		if planner.SargableFor(where, keys) == 0 {
			// Index not applicable
			continue
		}

		indexCond := index.Condition()
		if indexCond == nil {
			unfiltered[index] = keys
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

		if where.EquivalentTo(indexCond) {
			// Index condition equivalent to query condition
			equivalent[index] = keys
			break
		}

		if planner.SubsetOf(where, indexCond) {
			// Index condition satisfies query condition
			filtered[index] = keys
		}
	}

	var indexMap map[datastore.Index]expression.Expressions
	if len(equivalent) > 0 {
		indexMap = equivalent
	} else if len(filtered) > 0 {
		indexMap = filtered
	} else if len(unfiltered) > 0 {
		indexMap = unfiltered
	}

	for index, keys := range indexMap {
		spans, err := planner.SargFor(where, keys)
		if len(spans) == 0 {
			logging.Errorp("Sargable index not sarged", logging.Pair{"where", where},
				logging.Pair{"index_keys", keys}, logging.Pair{"error", err})
			return nil, errors.NewPlanError(nil, fmt.Sprintf("Sargable index not sarged; where=%v, index_keys=%v, error=%v",
				where.String(), keys.String(), err))
		}

		var scan Operator
		scan = NewIndexScan(index, node, spans, false, math.MaxInt64)
		if len(spans) > 1 {
			// Use UnionScan to de-dup multiple spans
			scan = NewUnionScan(scan)
		}

		return scan, err
	}

	return this.selectPrimaryScan(keyspace, node, primaryIndexes)
}

func (this *builder) selectPrimaryScan(keyspace datastore.Keyspace,
	node *algebra.KeyspaceTerm, primaryIndexes map[datastore.PrimaryIndex]bool) (Operator, error) {
	var primary datastore.PrimaryIndex

	// Prefer any primary indexes from hints
	for index, _ := range primaryIndexes {
		state, _, er := index.State()
		if er != nil {
			logging.Errorp("PrimaryScan Selection", logging.Pair{"error", er.Error()})
		}

		if state != datastore.ONLINE {
			primary = index
			continue
		}

		scan := NewPrimaryScan(index, keyspace, node)
		return scan, nil
	}

	// Now consider all primary indexes

	indexers, err := keyspace.Indexers()
	if err != nil {
		return nil, err
	}

	for _, indexer := range indexers {
		indexes, err := indexer.PrimaryIndexes()
		if err != nil {
			return nil, err
		}

		for _, index := range indexes {
			state, _, er := index.State()
			if er != nil {
				logging.Errorp("PrimaryScan Selection", logging.Pair{"error", er.Error()})
			}

			if state != datastore.ONLINE {
				primary = index
				continue
			}

			scan := NewPrimaryScan(index, keyspace, node)
			return scan, nil
		}
	}

	if primary == nil {
		return nil, fmt.Errorf(
			"No primary index on keyspace %s. Use CREATE PRIMARY INDEX to create one.",
			keyspace.Name())
	}

	return nil, fmt.Errorf("Primary index %s not online.", primary.Name())
}
