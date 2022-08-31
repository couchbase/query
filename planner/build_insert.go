//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/plan"
)

func (this *builder) VisitInsert(stmt *algebra.Insert) (interface{}, error) {
	ksref := stmt.KeyspaceRef()
	ksref.SetDefaultNamespace(this.namespace)

	keyspace, err := this.getNameKeyspace(ksref, true)
	if err != nil {
		return nil, err
	}

	if keyspace != nil {
		this.skipKeyspace = keyspace.QualifiedName()
	}

	children := make([]plan.Operator, 0, 4)

	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL
	size := OPT_SIZE_NOT_AVAIL
	frCost := OPT_COST_NOT_AVAIL

	qp := plan.NewQueryPlan(nil)
	err = this.chkBldSubqueries(stmt, qp)
	if err != nil {
		return nil, err
	}

	if stmt.Values() != nil {
		if this.useCBO && this.keyspaceUseCBO(ksref.Alias()) {
			cost, cardinality, size, frCost = getValueScanCost(stmt.Values())
		}
		children = append(children, plan.NewValueScan(stmt.Values(), cost, cardinality, size, frCost))
		this.maxParallelism = (len(stmt.Values()) + 64) / 64
	} else if stmt.Select() != nil {
		q, err := stmt.Select().Accept(this)
		if err != nil {
			return nil, err
		}

		selQP := q.(*plan.QueryPlan)
		selOp := selQP.PlanOp()
		if this.useCBO {
			cost = selOp.Cost()
			cardinality = selOp.Cardinality()
			size = selOp.Size()
			frCost = selOp.FrCost()
		}
		children = append(children, selOp)
	} else {
		return nil, fmt.Errorf("INSERT missing both VALUES and SELECT.")
	}

	if this.useCBO && cost > 0.0 && cardinality > 0.0 && size > 0 && frCost > 0.0 {
		cost, cardinality, size, frCost = getInsertCost(stmt.Key(), stmt.Value(),
			stmt.Options(), nil, cost, cardinality, size, frCost)
	}

	insert := plan.NewSendInsert(keyspace, ksref, stmt.Key(), stmt.Value(), stmt.Options(),
		nil, cost, cardinality, size, frCost, this.mustSkipKeys)
	subChildren := make([]plan.Operator, 0, 4)
	subChildren = append(subChildren, insert)

	if stmt.Returning() != nil {
		subChildren = this.buildDMLProject(stmt.Returning(), subChildren)
	} else {
		subChildren = append(subChildren, plan.NewDiscard(cost, cardinality, size, frCost))
	}

	children = append(children, this.addParallel(subChildren...))
	qp.SetPlanOp(plan.NewSequence(children...))
	return qp, nil
}
