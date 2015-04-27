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
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
)

func BuildScan(keyspace datastore.Keyspace, node *algebra.KeyspaceTerm, pred expression.Expression) (
	secondary map[datastore.Index]Spans, primary datastore.PrimaryIndex, err error) {
	var indexes, hintIndexes, otherIndexes []datastore.Index
	hints := node.Indexes()
	if hints != nil {
		indexes, err = allHints(keyspace, hints)
		hintIndexes = indexes
	} else {
		indexes, err = allIndexes(keyspace)
		otherIndexes = indexes
	}

	if err != nil {
		return
	}

	if pred != nil {
		nnf := NewNNF()
		pred = pred.Copy()
		pred, err = nnf.Map(pred)
		if err != nil {
			return
		}

		formalizer := expression.NewFormalizer()
		formalizer.Keyspace = node.Alias()
		primaryKey := expression.Expressions{
			expression.NewField(
				expression.NewMeta(expression.NewConstant(node.Alias())),
				expression.NewFieldName("id")),
		}

		sargables, er := sargableIndexes(indexes, pred, primaryKey, nnf, formalizer)
		if er != nil {
			return nil, nil, er
		}

		minimals, er := minimalIndexes(sargables, pred)
		if er != nil {
			return nil, nil, er
		}

		if len(minimals) > 0 {
			return minimals, nil, nil
		}
	}

	primary, err = buildPrimaryScan(keyspace, hintIndexes, otherIndexes)
	if err != nil {
		return nil, nil, err
	}

	return
}

func allHints(keyspace datastore.Keyspace, hints algebra.IndexRefs) ([]datastore.Index, error) {
	indexes := make([]datastore.Index, 0, len(hints))

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

func allIndexes(keyspace datastore.Keyspace) ([]datastore.Index, error) {
	indexers, err := keyspace.Indexers()
	if err != nil {
		return nil, err
	}

	indexes := make([]datastore.Index, 0, len(indexers)*16)

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

type indexEntry struct {
	keys expression.Expressions
	cond expression.Expression
}

func sargableIndexes(indexes []datastore.Index, pred expression.Expression,
	primaryKey expression.Expressions, nnf *NNF, formalizer *expression.Formalizer) (
	map[datastore.Index]*indexEntry, error) {
	var err error
	var keys expression.Expressions
	sargables := make(map[datastore.Index]*indexEntry, len(indexes))

	for _, index := range indexes {
		if index.IsPrimary() {
			keys = primaryKey
		} else {
			keys = index.RangeKey()
			keys = keys.Copy()

			for i, key := range keys {
				key = key.Copy()

				key, err = formalizer.Map(key)
				if err != nil {
					return nil, err
				}

				key, err = nnf.Map(key)
				if err != nil {
					return nil, err
				}

				keys[i] = key
			}
		}

		cond := index.Condition()
		if cond != nil {
			cond = cond.Copy()

			cond, err = formalizer.Map(cond)
			if err != nil {
				return nil, err
			}

			cond, err = nnf.Map(cond)
			if err != nil {
				return nil, err
			}

			if !SubsetOf(pred, cond) {
				continue
			}
		}

		n := SargableFor(pred, keys)
		if n > 0 {
			sargables[index] = &indexEntry{keys[0:n], cond}
		}
	}

	return sargables, nil
}

func minimalIndexes(sargables map[datastore.Index]*indexEntry, pred expression.Expression) (
	map[datastore.Index]Spans, error) {
	for s, se := range sargables {
		for t, te := range sargables {
			if t == s {
				continue
			}

			if narrowerOrEquivalent(se, te) {
				delete(sargables, t)
			}
		}
	}

	minimals := make(map[datastore.Index]Spans, len(sargables))
	for s, se := range sargables {
		spans, err := SargFor(pred, se.keys)
		if err != nil || len(spans) == 0 {
			logging.Errorp("Sargable index not sarged", logging.Pair{"pred", pred},
				logging.Pair{"index_keys", se.keys}, logging.Pair{"error", err})
			return nil, errors.NewPlanError(nil, fmt.Sprintf("Sargable index not sarged; pred=%v, index_keys=%v, error=%v",
				pred.String(), se.keys.String(), err))
			return nil, err
		}

		minimals[s] = spans
	}

	return minimals, nil
}

func narrowerOrEquivalent(se, te *indexEntry) bool {
	if len(te.keys) > len(se.keys) {
		return false
	}

	if te.cond != nil && (se.cond == nil || !SubsetOf(se.cond, te.cond)) {
		return false
	}

outer:
	for _, tk := range te.keys {
		for _, sk := range se.keys {
			if SubsetOf(sk, tk) {
				continue outer
			}
		}

		return false
	}

	return true
}

func buildPrimaryScan(keyspace datastore.Keyspace, hintIndexes, otherIndexes []datastore.Index) (
	primary datastore.PrimaryIndex, err error) {
	ok := false

	// Prefer hints
	for _, index := range hintIndexes {
		primary, ok = index.(datastore.PrimaryIndex)
		if ok {
			return
		}
	}

	// Consider other primary indexes
	if otherIndexes != nil {
		for _, index := range otherIndexes {
			primary, ok = index.(datastore.PrimaryIndex)
			if ok {
				return
			}
		}
	}

	// Return first online primary index
	indexers, er := keyspace.Indexers()
	if er != nil {
		return nil, err
	}

	for _, indexer := range indexers {
		primaries, er := indexer.PrimaryIndexes()
		if er != nil {
			return nil, er
		}

		for _, primary = range primaries {
			state, _, er := primary.State()
			if er != nil {
				return nil, er
			}

			if state == datastore.ONLINE {
				return
			}
		}
	}

	if primary == nil {
		return nil, fmt.Errorf(
			"No primary index on keyspace %s. Use CREATE PRIMARY INDEX to create one.",
			keyspace.Name())
	}

	return nil, fmt.Errorf("Primary index %s not online.", primary.Name())
}
