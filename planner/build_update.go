//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
)

func (this *builder) VisitUpdate(stmt *algebra.Update) (interface{}, error) {
	this.where = stmt.Where()
	this.node = stmt

	this.initialIndexAdvisor(stmt)
	ksref := stmt.KeyspaceRef()
	keyspace, err := this.getNameKeyspace(ksref, true)
	if err != nil {
		return nil, err
	}

	qp := plan.NewQueryPlan(nil)
	err = this.chkBldSubqueries(stmt, qp)
	if err != nil {
		return nil, err
	}

	optimHints := stmt.OptimHints()
	optimHints, err = this.beginMutate(keyspace, ksref, stmt.Keys(), stmt.Indexes(), stmt.Limit(), nil, true, optimHints,
		stmt.Let())
	if err != nil {
		return nil, err
	}
	stmt.SetOptimHints(optimHints)

	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL
	size := OPT_SIZE_NOT_AVAIL
	frCost := OPT_COST_NOT_AVAIL
	if this.useCBO && this.lastOp != nil {
		cost = this.lastOp.Cost()
		cardinality = this.lastOp.Cardinality()
		size = this.lastOp.Size()
		frCost = this.lastOp.FrCost()
		if cost > 0.0 && cardinality > 0.0 && size > 0 && frCost > 0.0 {
			cost, cardinality, size, frCost = getCloneCost(cost, cardinality, size, frCost)
		}
	}

	subChildren := this.subChildren
	updateSubChildren := make([]plan.Operator, 0, 8)
	updateSubChildren = append(updateSubChildren, plan.NewClone(ksref.Alias(),
		cost, cardinality, size, frCost))

	if stmt.Set() != nil {
		if this.useCBO && cost > 0.0 && cardinality > 0.0 && size > 0 && frCost > 0.0 {
			cost, cardinality, size, frCost = getUpdateSetCost(stmt.Set(),
				cost, cardinality, size, frCost)
		}
		updateSubChildren = append(updateSubChildren, plan.NewSet(stmt.Set(),
			cost, cardinality, size, frCost))
	}

	if stmt.Unset() != nil {
		if this.useCBO && cost > 0.0 && cardinality > 0.0 && size > 0 && frCost > 0.0 {
			cost, cardinality, size, frCost = getUpdateUnsetCost(stmt.Unset(),
				cost, cardinality, size, frCost)
		}
		updateSubChildren = append(updateSubChildren, plan.NewUnset(stmt.Unset(),
			cost, cardinality, size, frCost))
	}

	if this.useCBO && cost > 0.0 && cardinality > 0.0 && size > 0 && frCost > 0.0 {
		cost, cardinality, size, frCost = getUpdateSendCost(stmt.Limit(),
			cost, cardinality, size, frCost)
	}
	updateSubChildren = append(updateSubChildren, plan.NewSendUpdate(keyspace, ksref, stmt.Limit(),
		cost, cardinality, size, frCost, stmt.Returning() == nil))

	if stmt.Returning() != nil {
		updateSubChildren = this.buildDMLProject(stmt.Returning(), updateSubChildren, true)
	}

	if stmt.Limit() != nil {
		seqChildren := make([]plan.Operator, 0, 3)
		seqChildren = append(seqChildren, this.addParallel(subChildren...))
		if this.useCBO && cost > 0.0 && cardinality > 0.0 && size > 0 && frCost > 0.0 {
			nlimit := int64(0)
			lv, static := base.GetStaticInt(stmt.Limit())
			if static {
				nlimit = lv
			}
			cost, cardinality, size, frCost = getLimitCost(this.lastOp, nlimit, -1)
		}
		seqChildren = append(seqChildren, plan.NewLimit(stmt.Limit(), cost, cardinality, size, frCost))
		seqChildren = append(seqChildren, this.addParallel(updateSubChildren...))
		this.addChildren(plan.NewSequence(seqChildren...))
	} else {
		subChildren = append(subChildren, updateSubChildren...)
		this.addChildren(this.addParallel(subChildren...))
	}

	qp.SetPlanOp(plan.NewSequence(this.children...))
	return qp, nil
}
