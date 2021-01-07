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

	err = this.beginMutate(keyspace, ksref, stmt.Keys(), stmt.Indexes(), stmt.Limit(), true)
	if err != nil {
		return nil, err
	}

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
		cost, cardinality, size, frCost))

	if stmt.Returning() != nil {
		updateSubChildren = this.buildDMLProject(stmt.Returning(), updateSubChildren)
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
			cost, cardinality, size, frCost = getLimitCost(this.lastOp, nlimit)
		}
		seqChildren = append(seqChildren, plan.NewLimit(stmt.Limit(), cost, cardinality, size, frCost))
		seqChildren = append(seqChildren, this.addParallel(updateSubChildren...))
		this.addChildren(plan.NewSequence(seqChildren...))
	} else {
		subChildren = append(subChildren, updateSubChildren...)
		this.addChildren(this.addParallel(subChildren...))
	}

	if stmt.Returning() == nil {
		this.addChildren(plan.NewDiscard(cost, cardinality, size, frCost))
	}

	return plan.NewSequence(this.children...), nil
}
