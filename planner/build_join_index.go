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
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
)

func (this *builder) buildIndexJoin(keyspace datastore.Keyspace,
	node *algebra.IndexJoin) (op *plan.IndexJoin, err error) {
	index, covers, err := this.buildJoinScan(keyspace, node.Right(), "join")
	if err != nil {
		return nil, err
	}

	return plan.NewIndexJoin(keyspace, node, index, covers), nil
}

func (this *builder) buildIndexNest(keyspace datastore.Keyspace,
	node *algebra.IndexNest) (op *plan.IndexNest, err error) {
	index, covers, err := this.buildJoinScan(keyspace, node.Right(), "nest")
	if err != nil {
		return nil, err
	}

	return plan.NewIndexNest(keyspace, node, index, covers), nil
}

func (this *builder) buildJoinScan(keyspace datastore.Keyspace, node *algebra.KeyspaceTerm, op string) (
	datastore.Index, expression.Covers, error) {
	indexes, err := allIndexes(keyspace)
	if err != nil {
		return nil, nil, err
	}

	var pred expression.Expression
	pred = expression.NewIsNotNull(node.Keys().Copy())
	dnf := NewDNF()
	pred, err = dnf.Map(pred)
	if err != nil {
		return nil, nil, err
	}

	subset := pred
	if this.where != nil {
		subset = expression.NewAnd(subset, this.where.Copy())
		subset, err = dnf.Map(subset)
		if err != nil {
			return nil, nil, err
		}
	}

	formalizer := expression.NewFormalizer(node.Alias(), nil)
	primaryKey := expression.Expressions{
		expression.NewField(
			expression.NewMeta(expression.NewConstant(node.Alias())),
			expression.NewFieldName("id", false)),
	}

	sargables, err := sargableIndexes(indexes, pred, subset, primaryKey, dnf, formalizer)
	if err != nil {
		return nil, nil, err
	}

	minimals, err := minimalIndexes(sargables, pred)
	if err != nil {
		return nil, nil, err
	}

	if len(minimals) == 0 {
		return nil, nil, errors.NewNoIndexJoinError(node.Alias(), op)
	}

	return this.buildCoveringJoinScan(minimals, node, op)
}

func (this *builder) buildCoveringJoinScan(secondaries map[datastore.Index]*indexEntry,
	node *algebra.KeyspaceTerm, op string) (datastore.Index, expression.Covers, error) {
	if this.cover == nil {
		for index, _ := range secondaries {
			return index, nil, nil
		}
	}

	alias := node.Alias()
	exprs := this.cover.Expressions()

outer:
	for index, entry := range secondaries {
		for _, expr := range exprs {
			if !expr.CoveredBy(alias, entry.keys) {
				continue outer
			}
		}

		covers := make(expression.Covers, len(entry.keys))
		for i, key := range entry.keys {
			covers[i] = expression.NewCover(key)
		}

		return index, covers, nil
	}

	for index, _ := range secondaries {
		return index, nil, nil
	}

	return nil, nil, errors.NewNoIndexJoinError(node.Alias(), op)
}
