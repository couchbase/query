//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

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
	baseKeyspace := base.NewBaseKeyspace(ksref.Alias(), ksref.Path(), term, 1)
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
	size := scan.Size()
	frCost := scan.FrCost()

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
			if this.useCBO && (cost > 0.0) && (size > 0) && (frCost > 0.0) {
				fetchCost, fsize, ffrCost := getFetchCost(keyspace, cardinality)
				if fetchCost > 0.0 && fsize > 0 && ffrCost > 0.0 {
					cost += fetchCost
					frCost += ffrCost
					size = fsize
				} else {
					cost = OPT_COST_NOT_AVAIL
					cardinality = OPT_CARD_NOT_AVAIL
					size = OPT_SIZE_NOT_AVAIL
					frCost = OPT_COST_NOT_AVAIL
				}
			}
			fetch = plan.NewFetch(keyspace, term, names, cost, cardinality, size, frCost)
		} else {
			fetch = plan.NewDummyFetch(keyspace, term, cost, cardinality, size, frCost)
		}
		this.addSubChildren(fetch)
	}

	if this.where != nil {
		if this.useCBO {
			cost, cardinality, size, frCost = getFilterCost(this.lastOp, this.where,
				this.baseKeyspaces, this.keyspaceNames, term.Alias(),
				this.advisorValidate(), this.context)
		}

		filter := plan.NewFilter(this.where, cost, cardinality, size, frCost)
		this.addSubChildren(filter)
	}

	return nil
}

func isKeyScan(scan plan.Operator) bool {
	_, rv := scan.(*plan.KeyScan)
	return rv
}
