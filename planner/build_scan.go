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
)

func (this *builder) selectScan(keyspace datastore.Keyspace, node *algebra.KeyspaceTerm) (op plan.Operator, err error) {

	keys := node.Keys()
	if keys != nil {
		this.resetOrderOffsetLimit()
		switch keys.(type) {
		case *expression.ArrayConstruct, *algebra.NamedParameter, *algebra.PositionalParameter:
			this.maxParallelism = 0
		default:
			this.maxParallelism = 1
		}

		return plan.NewKeyScan(keys), nil
	}

	this.maxParallelism = 0 // Use default parallelism for index scans

	secondary, primary, err := this.buildScan(keyspace, node)
	if err != nil {
		return nil, err
	}

	if secondary != nil {
		return secondary, nil
	} else {
		return primary, nil
	}
}

func (this *builder) buildScan(keyspace datastore.Keyspace, node *algebra.KeyspaceTerm) (
	secondary plan.Operator, primary *plan.PrimaryScan, err error) {

	var hints []datastore.Index
	if len(node.Indexes()) > 0 {
		hints = _HINT_POOL.Get()
		defer _HINT_POOL.Put(hints)
		hints, err = allHints(keyspace, node.Indexes(), hints)
		if err != nil {
			return
		}
	}

	pred := this.where
	if pred != nil {
		// Handle constant TRUE predicate
		cpred := pred.Value()
		if cpred != nil && cpred.Truth() {
			pred = nil
		}
	}

	id := expression.NewField(
		expression.NewMeta(expression.NewIdentifier(node.Alias())),
		expression.NewFieldName("id", false))

	// First handle covering primary scan
	if this.cover != nil && pred == nil {
		scan, err := this.buildCoveringPrimaryScan(keyspace, node, id, hints)
		if scan != nil || err != nil {
			return scan, nil, err
		}
	}

	if pred != nil {
		return this.buildPredicateScan(keyspace, node, id, pred, hints)
	}

	if this.order != nil {
		this.resetOrderOffsetLimit()
	}

	primary, err = this.buildPrimaryScan(keyspace, node, hints, false, true)
	return nil, primary, err
}

func (this *builder) buildPredicateScan(keyspace datastore.Keyspace, node *algebra.KeyspaceTerm,
	id, pred expression.Expression, hints []datastore.Index) (
	secondary plan.Operator, primary *plan.PrimaryScan, err error) {

	// Handle constant FALSE predicate
	cpred := pred.Value()
	if cpred != nil && !cpred.Truth() {
		return _EMPTY_PLAN, nil, nil
	}

	primaryKey := expression.Expressions{id}
	formalizer := expression.NewSelfFormalizer(node.Alias(), nil)

	if len(hints) > 0 {
		secondary, primary, err = this.buildSubsetScan(
			keyspace, node, id, pred, hints, primaryKey, formalizer, true)
		if secondary != nil || primary != nil || err != nil {
			return
		}
	}

	others := _INDEX_POOL.Get()
	defer _INDEX_POOL.Put(others)
	others, err = allIndexes(keyspace, hints, others)
	if err != nil {
		return
	}

	return this.buildSubsetScan(keyspace, node, id, pred, others, primaryKey, formalizer, false)
}

func (this *builder) buildSubsetScan(keyspace datastore.Keyspace, node *algebra.KeyspaceTerm,
	id, pred expression.Expression, indexes []datastore.Index,
	primaryKey expression.Expressions, formalizer *expression.Formalizer, force bool) (
	secondary plan.Operator, primary *plan.PrimaryScan, err error) {

	// Consider pattern matching indexes
	pred, err = PatternFor(pred, indexes, formalizer)
	if err != nil {
		return nil, nil, err
	}

	// Prefer OR scan
	if or, ok := pred.(*expression.Or); ok {
		scan, _, err := this.buildOrScan(node, id, or, indexes, primaryKey, formalizer)
		if scan != nil || err != nil {
			return scan, nil, err
		}
	}

	// Prefer secondary scan
	secondary, _, err = this.buildTermScan(node, id, pred, indexes, primaryKey, formalizer)
	if secondary != nil || err != nil {
		return secondary, nil, err
	}

	// No secondary scan, try primary scan
	primary, err = this.buildPrimaryScan(keyspace, node, indexes, force, false)
	if err != nil {
		return nil, nil, err
	}

	// Primary scan with predicates -- disable pushdown
	if primary != nil {
		this.resetPushDowns()
	}

	return nil, primary, nil
}

func (this *builder) buildTermScan(node *algebra.KeyspaceTerm,
	id, pred expression.Expression, indexes []datastore.Index,
	primaryKey expression.Expressions, formalizer *expression.Formalizer) (
	secondary plan.SecondaryScan, sargLength int, err error) {

	var scanbuf [4]plan.SecondaryScan
	scans := scanbuf[0:1]

	dnfPred := pred.Copy()
	dnf := NewDNF(dnfPred, true)
	dnfPred, err = dnf.Map(dnfPred)
	if err != nil {
		return nil, 0, err
	}

	sargables, all, arrays, err := sargableIndexes(indexes, dnfPred, dnfPred, primaryKey, formalizer)
	if err != nil {
		return nil, 0, err
	}

	minimals := minimalIndexes(sargables, false)

	order := this.order
	limitExpr := this.limit
	offsetExpr := this.offset
	countAgg := this.countAgg
	countDistinctAgg := this.countDistinctAgg
	minAgg := this.minAgg
	maxAgg := this.maxAgg
	defer func() {
		if this.orderScan != nil {
			this.order = order
		}
	}()
	var secOffsetPushed, unnestOffsetPushed, dynamicOffsetPushed bool
	var limitPushed bool

	// Try secondary scan
	if len(minimals) > 0 {
		secondary, sargLength, err = this.buildSecondaryScan(minimals, node, id, dnfPred)
		if err != nil {
			return nil, 0, err
		}

		if secondary != nil {
			if len(this.coveringScans) > 0 || this.countScan != nil {
				return secondary, sargLength, nil
			}

			if secondary == this.orderScan {
				scans[0] = secondary
			} else {
				scans = append(scans, secondary)
			}

			secOffsetPushed = this.offset != nil
			limitPushed = limitPushed || this.limit != nil
		}
	}

	// Try UNNEST scan
	if this.from != nil {
		// Try pushdowns
		this.order = order
		this.limit = limitExpr
		this.offset = offsetExpr
		this.countAgg = countAgg
		this.countDistinctAgg = countDistinctAgg
		this.minAgg = minAgg
		this.maxAgg = maxAgg

		unnest, unnestSargLength, err := this.buildUnnestScan(node, this.from, dnfPred, all)
		if err != nil {
			return nil, 0, err
		}

		if unnest != nil {
			if len(this.coveringScans) > 0 || this.countScan != nil {
				return unnest, unnestSargLength, err
			}

			scans = append(scans, unnest)
			if sargLength < unnestSargLength {
				sargLength = unnestSargLength
			}

			unnestOffsetPushed = this.offset != nil
			limitPushed = limitPushed || this.limit != nil
		}

		this.resetPushDowns()
	}

	// Try dynamic scan
	if len(arrays) > 0 {
		// Try pushdowns
		this.limit = limitExpr
		this.offset = offsetExpr

		dynamicPred := pred.Copy()
		dnf := NewDNF(dynamicPred, false)
		dynamicPred, err = dnf.Map(dynamicPred)
		if err != nil {
			return nil, 0, err
		}

		dynamic, dynamicSargLength, err :=
			this.buildDynamicScan(node, id, dynamicPred, arrays, primaryKey, formalizer)
		if err != nil {
			return nil, 0, err
		}

		if dynamic != nil {
			if len(this.coveringScans) > 0 || this.countScan != nil {
				return dynamic, dynamicSargLength, err
			}

			scans = append(scans, dynamic)
			if sargLength < dynamicSargLength {
				sargLength = dynamicSargLength
			}
			dynamicOffsetPushed = this.offset != nil
			limitPushed = limitPushed || this.limit != nil
		}
	}

	switch len(scans) {
	case 0:
		this.limit = limitExpr
		this.offset = offsetExpr
		secondary = nil
	case 1:
		this.resetOffset()
		if secOffsetPushed || unnestOffsetPushed || dynamicOffsetPushed {
			this.offset = offsetExpr
		}
		secondary = scans[0]
	default:
		this.resetOffset()
		var limit expression.Expression

		if limitPushed {
			limit = offsetPlusLimit(offsetExpr, limitExpr)
		}

		if scans[0] == nil {
			if len(scans) == 2 {
				if secOffsetPushed || unnestOffsetPushed || dynamicOffsetPushed {
					this.offset = offsetExpr
				}
				secondary = scans[1]
			} else {
				secondary = plan.NewIntersectScan(limit, scans[1:]...)
			}
		} else {
			if ordered, ok := scans[0].(*plan.OrderedIntersectScan); ok {
				scans = append(ordered.Scans(), scans[1:]...)
			}

			secondary = plan.NewOrderedIntersectScan(limit, scans...)
		}
	}

	// Return secondary scan, if any
	return secondary, sargLength, nil
}

func allHints(keyspace datastore.Keyspace, hints algebra.IndexRefs, indexes []datastore.Index) (
	[]datastore.Index, error) {

	for _, hint := range hints {
		indexer, err := keyspace.Indexer(hint.Using())
		if err != nil {
			return nil, err
		}

		// refresh indexer
		_, err = indexer.Indexes()
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

func allIndexes(keyspace datastore.Keyspace, skip, indexes []datastore.Index) (
	[]datastore.Index, error) {

	indexers, err := keyspace.Indexers()
	if err != nil {
		return nil, err
	}

	var skipMap map[datastore.Index]bool
	if len(skip) > 0 {
		skipMap = _SKIP_POOL.Get()
		defer _SKIP_POOL.Put(skipMap)
		for _, s := range skip {
			skipMap[s] = true
		}
	}

	for _, indexer := range indexers {
		idxes, err := indexer.Indexes()
		if err != nil {
			return nil, err
		}

		for _, idx := range idxes {
			// Skip index if listed
			if len(skipMap) > 0 && skipMap[idx] {
				continue
			}

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

var _INDEX_POOL = datastore.NewIndexPool(256)
var _HINT_POOL = datastore.NewIndexPool(32)
var _SKIP_POOL = datastore.NewIndexBoolPool(32)
var _EMPTY_PLAN = plan.NewValueScan(algebra.Pairs{})
