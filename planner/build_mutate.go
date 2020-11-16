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
	base "github.com/couchbase/query/plannerbase"
)

func (this *builder) beginMutate(keyspace datastore.Keyspace, ksref *algebra.KeyspaceRef,
	keys expression.Expression, indexes algebra.IndexRefs, limit expression.Expression, mustFetch bool) error {
	ksref.SetDefaultNamespace(this.namespace)
	var term *algebra.KeyspaceTerm
	if ksref.Path() != nil {
		term = algebra.NewKeyspaceTermFromPath(ksref.Path(), ksref.As(), keys, indexes)
	} else {
		term = algebra.NewKeyspaceTermFromExpression(ksref.ExpressionTerm(), ksref.As(), keys, indexes, 0)
	}

	this.children = make([]plan.Operator, 0, 8)
	this.subChildren = make([]plan.Operator, 0, 8)

	prevLimit := this.limit
	prevOffset := this.offset
	prevRequirePrimaryKey := this.requirePrimaryKey
	prevBasekeyspaces := this.baseKeyspaces
	prevCover := this.cover

	defer func() {
		this.offset = prevOffset
		this.cover = prevCover
		this.limit = prevLimit
		this.requirePrimaryKey = prevRequirePrimaryKey
		this.baseKeyspaces = prevBasekeyspaces
	}()

	this.limit = limit
	this.offset = nil
	this.requirePrimaryKey = true
	this.baseKeyspaces = make(map[string]*base.BaseKeyspace, _MAP_KEYSPACE_CAP)
	baseKeyspace := base.NewBaseKeyspace(ksref.Alias(), ksref.Path())
	this.baseKeyspaces[baseKeyspace.Name()] = baseKeyspace
	this.collectKeyspaceNames()
	this.extractKeyspacePredicates(this.where, nil)

	if !mustFetch {
		mustFetch = this.context.HasDeltaKeyspace(baseKeyspace.Keyspace())
	}
	if mustFetch {
		this.cover = nil
	}

	// Process where clause
	if this.where != nil {
		err := this.processWhere(this.where)
		if err != nil {
			return err
		}
	}

	scan, err := this.selectScan(keyspace, term, true)

	this.appendQueryInfo(scan, keyspace, term, len(this.coveringScans) == 0)

	if err != nil {
		return err
	}

	this.addChildren(scan)

	cost := scan.Cost()
	cardinality := scan.Cardinality()

	if len(this.coveringScans) > 0 {
		err = this.coverExpressions()
		if err != nil {
			return err
		}
	} else {
		var fetch plan.Operator
		if mustFetch || this.where != nil || !isKeyScan(scan) {
			names, err := this.GetSubPaths(term.Alias())
			if err != nil {
				return err
			}
			if this.useCBO && (cost > 0.0) {
				fetchCost := getFetchCost(keyspace, cardinality)
				if fetchCost > 0.0 {
					cost += fetchCost
				} else {
					cost = OPT_COST_NOT_AVAIL
					cardinality = OPT_CARD_NOT_AVAIL
				}
			}
			fetch = plan.NewFetch(keyspace, term, names, cost, cardinality)
		} else {
			fetch = plan.NewDummyFetch(keyspace, term, cost, cardinality)
		}
		this.addSubChildren(fetch)
	}

	if this.where != nil {
		if this.useCBO {
			cost, cardinality = getFilterCost(this.lastOp, this.where,
				this.baseKeyspaces, this.keyspaceNames, term.Alias(),
				this.advisorValidate(), this.context)
		}

		filter := plan.NewFilter(this.where, cost, cardinality)
		this.addSubChildren(filter)
	}

	return nil
}

func isKeyScan(scan plan.Operator) bool {
	_, rv := scan.(*plan.KeyScan)
	return rv
}
