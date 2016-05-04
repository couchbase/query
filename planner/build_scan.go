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
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
)

func (this *builder) selectScan(keyspace datastore.Keyspace, node *algebra.KeyspaceTerm,
	limit expression.Expression) (op plan.Operator, err error) {
	keys := node.Keys()
	if keys != nil {
		this.resetOrderLimit()
		switch keys := keys.(type) {
		case *expression.ArrayConstruct:
			this.maxParallelism = util.MaxInt(1, len(keys.Operands()))
		case *algebra.NamedParameter, *algebra.PositionalParameter:
			this.maxParallelism = 0
		default:
			this.maxParallelism = 1
		}

		return plan.NewKeyScan(keys), nil
	}

	this.maxParallelism = 0 // Use default parallelism for index scans

	secondary, primary, err := this.buildScan(keyspace, node, limit)
	if err != nil {
		return nil, err
	}

	if secondary != nil {
		return secondary, nil
	} else {
		return primary, nil
	}
}

func (this *builder) buildScan(keyspace datastore.Keyspace, node *algebra.KeyspaceTerm, limit expression.Expression) (
	secondary plan.Operator, primary *plan.PrimaryScan, err error) {
	var indexes, hintIndexes, otherIndexes []datastore.Index
	hints := node.Indexes()
	if hints != nil {
		indexes = _INDEX_POOL.Get()
		defer _INDEX_POOL.Put(indexes)
		indexes, err = allHints(keyspace, hints, indexes)
		hintIndexes = indexes
	} else {
		indexes = _ALL_INDEX_POOL.Get()
		defer _ALL_INDEX_POOL.Put(indexes)
		indexes, err = allIndexes(keyspace, indexes)
		otherIndexes = indexes
	}

	if err != nil {
		return
	}

	// for Watson, restrict system keyspaces to primary scans
	isSystem := keyspace.NamespaceId() == "#system"

	pred := this.where
	if pred != nil && !isSystem {
		// Handle constant TRUE predicate
		cpred := pred.Value()
		if cpred != nil && cpred.Truth() {
			pred = nil
		}
	}

	id := expression.NewField(
		expression.NewMeta(expression.NewIdentifier(node.Alias())),
		expression.NewFieldName("id", false))

	// Handle covering primary scan
	if this.cover != nil && pred == nil && !isSystem {
		scan, err := this.buildCoveringPrimaryScan(keyspace, node, id, limit, hintIndexes, otherIndexes)
		if scan != nil || err != nil {
			return scan, nil, err
		}
	}

	if pred != nil && !isSystem {
		// Handle constant FALSE predicate
		cpred := pred.Value()
		if cpred != nil && !cpred.Truth() {
			return _EMPTY_PLAN, nil, nil
		}

		pred = pred.Copy()
		dnf := NewDNF(pred)
		pred, err = dnf.Map(pred)
		if err != nil {
			return
		}

		formalizer := expression.NewFormalizer(node.Alias(), nil)
		primaryKey := expression.Expressions{id}
		sargables, all, er := sargableIndexes(indexes, pred, pred, primaryKey, formalizer)
		if er != nil {
			return nil, nil, er
		}

		minimals, er := minimalIndexes(sargables, pred)
		if er != nil {
			return nil, nil, er
		}

		if limit != nil && len(minimals) == 0 {
			// PrimaryScan with predicates disable pushdown
			prevLimit := this.limit
			defer func() { this.limit = prevLimit }()
			this.limit = nil
			limit = nil
		}

		if len(minimals) == 0 {
			this.resetCountMin()
		}

		if len(minimals) > 0 {
			secondary, err = this.buildSecondaryScan(minimals, node, id, pred, limit)
			return secondary, nil, err
		}

		if this.from != nil {
			// Try for an UNNEST scan
			secondary, err = this.buildUnnestScan(node, this.from, pred, all)
			if secondary != nil || err != nil {
				return secondary, nil, err
			}
		}
	}

	if this.order != nil {
		this.resetOrderLimit()
		limit = nil
	}

	primary, err = this.buildPrimaryScan(keyspace, node, limit, hintIndexes, otherIndexes)
	return nil, primary, err
}

func allHints(keyspace datastore.Keyspace, hints algebra.IndexRefs, indexes []datastore.Index) ([]datastore.Index, error) {
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
			logging.Errorp("Index selection", logging.Pair{"error", er.Error()})
		}

		if er != nil || state != datastore.ONLINE {
			continue
		}

		indexes = append(indexes, index)
	}

	return indexes, nil
}

func allIndexes(keyspace datastore.Keyspace, indexes []datastore.Index) ([]datastore.Index, error) {
	indexers, err := keyspace.Indexers()
	if err != nil {
		return nil, err
	}

	for _, indexer := range indexers {
		idxes, err := indexer.Indexes()
		if err != nil {
			return nil, err
		}

		for _, idx := range idxes {
			state, _, er := idx.State()
			if er != nil {
				logging.Errorp("Index selection", logging.Pair{"error", er.Error()})
			}

			if er != nil || state != datastore.ONLINE {
				continue
			}

			indexes = append(indexes, idx)
		}
	}

	return indexes, nil
}

var _INDEX_POOL = datastore.NewIndexPool(32)
var _ALL_INDEX_POOL = datastore.NewIndexPool(256)
var _EMPTY_PLAN = plan.NewValueScan(algebra.Pairs{})
