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

func (this *builder) VisitDelete(stmt *algebra.Delete) (interface{}, error) {
	this.cover = stmt
	this.node = stmt
	this.where = stmt.Where()

	this.initialIndexAdvisor(stmt)

	ksref := stmt.KeyspaceRef()
	ksref.SetDefaultNamespace(this.namespace)
	keyspace, err := this.getNameKeyspace(ksref, true)
	if err != nil {
		return nil, err
	}

	qp := plan.NewQueryPlan(nil)
	err = this.chkBldSubqueries(stmt, qp)
	if err != nil {
		return nil, err
	}

	mustFetch := stmt.Returning() != nil || this.context.DeltaKeyspaces() != nil
	optimHints := stmt.OptimHints()
	optimHints, err = this.beginMutate(keyspace, ksref, stmt.Keys(), stmt.Indexes(), stmt.Limit(), stmt.Offset(),
		mustFetch, optimHints)
	if err != nil {
		return nil, err
	}
	stmt.SetOptimHints(optimHints)

	subChildren := this.subChildren
	deleteSubChildren := make([]plan.Operator, 0, 4)

	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL
	size := OPT_SIZE_NOT_AVAIL
	frCost := OPT_COST_NOT_AVAIL

	lastOp := this.lastOp

	if this.useCBO && this.keyspaceUseCBO(ksref.Alias()) && lastOp != nil {
		cost = lastOp.Cost()
		cardinality = lastOp.Cardinality()
		size = lastOp.Size()
		frCost = lastOp.FrCost()
		if cost > 0.0 && cardinality > 0.0 && size > 0 && frCost > 0.0 {
			cost, cardinality, size, frCost = getDeleteCost(stmt.Limit(),
				cost, cardinality, size, frCost)
		}
	}

	sd := plan.NewSendDelete(keyspace, ksref, stmt.Limit(), cost, cardinality, size, frCost, stmt.Returning() == nil)
	sd.SetValidateKeys(stmt.ValidateKeys())
	deleteSubChildren = append(deleteSubChildren, sd)

	if stmt.Returning() != nil {
		deleteSubChildren = this.buildDMLProject(stmt.Returning(), deleteSubChildren)
	}

	if stmt.Limit() != nil || stmt.Offset() != nil {
		seqChildren := make([]plan.Operator, 0, 4)
		nOffset := int64(-1)

		if len(subChildren) > 0 {
			seqChildren = append(seqChildren, this.addParallel(subChildren...))
		}

		// If offset clause is present and has not been pushed down to index - add Offset operator to plan
		if stmt.Offset() != nil && !this.hasBuilderFlag(BUILDER_OFFSET_PUSHDOWN) {
			// cost of Offset
			if this.useCBO && cost > 0.0 && cardinality > 0.0 && size > 0 && frCost > 0.0 {
				ov, static := base.GetStaticInt(stmt.Offset())
				if static {
					nOffset = ov
				} else {
					nOffset = 0
				}
				cost, cardinality, size, frCost = getOffsetCost(lastOp, nOffset)
			}
			offsetOp := plan.NewOffset(stmt.Offset(), cost, cardinality, size, frCost)
			seqChildren = append(seqChildren, offsetOp)
			lastOp = offsetOp
		}

		if stmt.Limit() != nil {
			if this.useCBO && cost > 0.0 && cardinality > 0.0 && size > 0 && frCost > 0.0 {
				nlimit := int64(0)
				lv, static := base.GetStaticInt(stmt.Limit())
				if static {
					nlimit = lv
				}
				cost, cardinality, size, frCost = getLimitCost(lastOp, nlimit, -1)
			}
			limitOp := plan.NewLimit(stmt.Limit(), cost, cardinality, size, frCost)
			seqChildren = append(seqChildren, limitOp)
			lastOp = limitOp
		}

		seqChildren = append(seqChildren, this.addParallel(deleteSubChildren...))
		this.addChildren(plan.NewSequence(seqChildren...))
	} else {
		if len(subChildren) > 0 {
			subChildren = append(subChildren, deleteSubChildren...)
		} else {
			subChildren = deleteSubChildren
		}
		this.addChildren(this.addParallel(subChildren...))
	}

	qp.SetPlanOp(plan.NewSequence(this.children...))
	return qp, nil
}
