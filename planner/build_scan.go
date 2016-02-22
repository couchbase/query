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
		indexes, err = allHints(keyspace, hints)
		hintIndexes = indexes
	} else {
		indexes, err = allIndexes(keyspace)
		otherIndexes = indexes
	}

	if err != nil {
		return
	}

	pred := this.where
	if pred != nil {
		dnf := NewDNF()
		pred = pred.Copy()
		pred, err = dnf.Map(pred)
		if err != nil {
			return
		}

		formalizer := expression.NewFormalizer(node.Alias(), nil)
		primaryKey := expression.Expressions{
			expression.NewField(
				expression.NewMeta(expression.NewIdentifier(node.Alias())),
				expression.NewFieldName("id", false)),
		}

		sargables, er := sargableIndexes(indexes, pred, pred, primaryKey, dnf, formalizer)
		if er != nil {
			return nil, nil, er
		}

		minimals, er := minimalIndexes(sargables, pred)
		if er != nil {
			return nil, nil, er
		}

		if limit != nil {
			if len(minimals) == 0 || !pred.IsLimitPushable() {
				// PrimaryScan with predicates disable limit pushdown
				// Predicate conatins expression that disallows limit pushdown
				prevLimit := this.limit
				defer func() { this.limit = prevLimit }()
				this.limit = nil
				limit = nil
			}
		}

		if len(minimals) > 0 {
			secondary, err = this.buildSecondaryScan(minimals, node, pred, limit)
			return secondary, nil, err
		}
	}

	if this.order != nil {
		this.resetOrderLimit()
		limit = nil
	}

	primary, err = this.buildPrimaryScan(keyspace, node, limit, hintIndexes, otherIndexes)
	return nil, primary, err
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
	keys     expression.Expressions
	sargKeys expression.Expressions
	cond     expression.Expression
	spans    plan.Spans
}

func sargableIndexes(indexes []datastore.Index, pred, subset expression.Expression,
	primaryKey expression.Expressions, dnf *DNF, formalizer *expression.Formalizer) (
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

				key, err = dnf.Map(key)
				if err != nil {
					return nil, err
				}

				keys[i] = key
			}
		}

		cond := index.Condition()
		if cond != nil {
			if subset == nil {
				continue
			}

			cond = cond.Copy()

			cond, err = formalizer.Map(cond)
			if err != nil {
				return nil, err
			}

			cond, err = dnf.Map(cond)
			if err != nil {
				return nil, err
			}

			if !SubsetOf(subset, cond) {
				continue
			}
		}

		n := SargableFor(pred, keys)
		if n > 0 {
			sargables[index] = &indexEntry{keys, keys[0:n], cond, nil}
		}
	}

	return sargables, nil
}

func minimalIndexes(sargables map[datastore.Index]*indexEntry, pred expression.Expression) (
	map[datastore.Index]*indexEntry, error) {
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

	minimals := make(map[datastore.Index]*indexEntry, len(sargables))
	for s, se := range sargables {
		spans, err := SargFor(pred, se.sargKeys, len(se.keys))
		if err != nil || len(spans) == 0 {
			logging.Errorp("Sargable index not sarged", logging.Pair{"pred", pred},
				logging.Pair{"sarg_keys", se.sargKeys}, logging.Pair{"error", err})
			return nil, errors.NewPlanError(nil, fmt.Sprintf("Sargable index not sarged; pred=%v, sarg_keys=%v, error=%v",
				pred.String(), se.sargKeys.String(), err))
			return nil, err
		}

		se.spans = spans
		minimals[s] = se
	}

	return minimals, nil
}

func narrowerOrEquivalent(se, te *indexEntry) bool {
	if len(te.sargKeys) > len(se.sargKeys) {
		return false
	}

	if te.cond != nil && (se.cond == nil || !SubsetOf(se.cond, te.cond)) {
		return false
	}

outer:
	for _, tk := range te.sargKeys {
		for _, sk := range se.sargKeys {
			if SubsetOf(sk, tk) {
				continue outer
			}
		}

		return false
	}

	return len(se.sargKeys) > len(te.sargKeys) ||
		len(se.keys) <= len(te.keys)
}

func (this *builder) buildSecondaryScan(secondaries map[datastore.Index]*indexEntry,
	node *algebra.KeyspaceTerm, pred, limit expression.Expression) (plan.Operator, error) {
	if this.cover != nil {
		scan, err := this.buildCoveringScan(secondaries, node, pred, limit)
		if scan != nil || err != nil {
			return scan, err
		}
	}

	if (this.order != nil || limit != nil) && len(secondaries) > 1 {
		// This makes InterSectionscan disable limit pushdown, don't use index order
		this.resetOrderLimit()
		limit = nil
	}
	if this.order != nil && this.maxParallelism > 1 {
		this.resetOrderLimit()
		limit = nil
	}

	scans := make([]plan.Operator, 0, len(secondaries))
	var op plan.Operator
	for index, entry := range secondaries {
		if this.order != nil && !this.useIndexOrder(entry, entry.keys) {
			this.resetOrderLimit()
			limit = nil
		}

		if limit != nil && !pred.CoveredBy(node.Alias(), entry.keys) {
			// Predicate is not covered by index keys disable limit pushdown
			this.limit = nil
			limit = nil
		}

		if this.order != nil {
			this.maxParallelism = 1
		}

		op = plan.NewIndexScan(index, node, entry.spans, false, limit, nil)
		if len(entry.spans) > 1 {
			// Use UnionScan to de-dup multiple spans
			op = plan.NewUnionScan(op)
		} else {
			// Use UnionScan to de-dup array index scans
			for _, sk := range entry.sargKeys {
				if isArray, _ := sk.IsArrayIndexKey(); isArray {
					op = plan.NewUnionScan(op)
					break
				}
			}
		}

		scans = append(scans, op)
	}

	if len(scans) > 1 {
		return plan.NewIntersectScan(scans...), nil
	} else {
		return scans[0], nil
	}
}

func (this *builder) buildPrimaryScan(keyspace datastore.Keyspace, node *algebra.KeyspaceTerm,
	limit expression.Expression, hintIndexes, otherIndexes []datastore.Index) (scan *plan.PrimaryScan, err error) {
	primary, err := buildPrimaryIndex(keyspace, hintIndexes, otherIndexes)
	if err != nil {
		return nil, err
	}

	return plan.NewPrimaryScan(primary, keyspace, node, limit), nil
}

func buildPrimaryIndex(keyspace datastore.Keyspace, hintIndexes, otherIndexes []datastore.Index) (
	primary datastore.PrimaryIndex, err error) {
	ok := false

	// Prefer hints
	for _, index := range hintIndexes {
		if !index.IsPrimary() {
			continue
		}

		primary, ok = index.(datastore.PrimaryIndex)
		if ok {
			return
		} else {
			return nil, fmt.Errorf("Unable to cast primary index %s", index.Name())
		}
	}

	// Consider other primary indexes
	if otherIndexes != nil {
		for _, index := range otherIndexes {
			if !index.IsPrimary() {
				continue
			}

			primary, ok = index.(datastore.PrimaryIndex)
			if ok {
				return
			} else {
				return nil, fmt.Errorf("Unable to cast primary index %s", index.Name())
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

func (this *builder) useIndexOrder(entry *indexEntry, keys expression.Expressions) bool {

	// If it makes UnionScan don't use index order
	if len(entry.spans) > 1 {
		return false
	} else {
		for _, sk := range entry.sargKeys {
			if isArray, _ := sk.IsArrayIndexKey(); isArray {
				return false
			}
		}
	}

	if len(keys) < len(this.order.Terms()) {
		return false
	}
	for i, orderterm := range this.order.Terms() {
		if orderterm.Descending() {
			return false
		}
		if !orderterm.Expression().EquivalentTo(keys[i]) {
			return false
		}
	}
	return true
}
