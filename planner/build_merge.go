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
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
)

func (this *builder) VisitMerge(stmt *algebra.Merge) (interface{}, error) {
	var path *algebra.Path

	this.node = stmt
	this.children = make([]plan.Operator, 0, 8)
	this.subChildren = make([]plan.Operator, 0, 8)
	source := stmt.Source()

	this.baseKeyspaces = make(map[string]*base.BaseKeyspace, _MAP_KEYSPACE_CAP)
	if source.From() != nil {
		path = source.From().Path()
	}
	sourceKeyspace := base.NewBaseKeyspace(source.Alias(), path)
	this.baseKeyspaces[sourceKeyspace.Name()] = sourceKeyspace
	targetKeyspace := base.NewBaseKeyspace(stmt.KeyspaceRef().Alias(), stmt.KeyspaceRef().Path())
	this.baseKeyspaces[targetKeyspace.Name()] = targetKeyspace
	this.collectKeyspaceNames()

	var left algebra.SimpleFromTerm
	var err error

	outer := false
	if stmt.Actions().Insert() != nil {
		// use outer join if INSERT action is specified
		outer = true
	}

	if !stmt.IsOnKey() && !outer {
		// setup usable predicate from ON-clause for source scan
		_, err = this.processPredicate(stmt.On(), true)
		if err != nil {
			return nil, err
		}

		this.pushableOnclause = stmt.On()
	}

	this.initialIndexAdvisor(stmt)
	this.extractKeyspacePredicates(nil, this.pushableOnclause)

	if source.SubqueryTerm() != nil {
		_, err = source.SubqueryTerm().Accept(this)
		if err != nil && !this.indexAdvisor {
			return nil, err
		}

		left = source.SubqueryTerm()
	} else if source.ExpressionTerm() != nil {
		_, err = source.ExpressionTerm().Accept(this)
		if err != nil && !this.indexAdvisor {
			return nil, err
		}

		left = source.ExpressionTerm()
	} else {
		if source.From() == nil {
			// should have caught in semantics check
			return nil, errors.NewPlanInternalError("VisitMerge: MERGE missing source.")
		}

		_, err = source.From().Accept(this)
		if err != nil && !this.indexAdvisor {
			return nil, err
		}

		left = source.From()
	}

	ksref := stmt.KeyspaceRef()
	ksref.SetDefaultNamespace(this.namespace)

	keyspace, err := this.getNameKeyspace(ksref, false)
	if err != nil {
		return nil, err
	}

	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL
	joinCost := OPT_COST_NOT_AVAIL
	joinCard := OPT_CARD_NOT_AVAIL
	leftCard := OPT_CARD_NOT_AVAIL

	if this.useCBO && this.lastOp != nil {
		leftCard = this.lastOp.Cardinality()
	}

	right := algebra.NewKeyspaceTermFromPath(ksref.Path(), ksref.As(), nil, stmt.Indexes())

	if stmt.IsOnKey() {
		if this.useCBO {
			leftKeyspaces, _, rightKeyspace, _ := this.getKeyspacesAliases(targetKeyspace.Name())
			joinCost, joinCard = getLookupJoinCost(this.lastOp, outer, right,
				leftKeyspaces, rightKeyspace)
		}
	} else {
		// use ANSI JOIN to handle the ON-clause
		right.SetAnsiJoin()
		algebra.TransferJoinHint(right, left)

		ansiJoin := algebra.NewAnsiJoin(left, outer, right, stmt.On())
		join, err := this.buildAnsiJoin(ansiJoin)
		if err != nil {
			return nil, err
		}

		switch join := join.(type) {
		case *plan.NLJoin:
			this.addSubChildren(join)
		case *plan.Join, *plan.HashJoin:
			if len(this.subChildren) > 0 {
				this.addChildren(this.addSubchildrenParallel())
			}
			this.addChildren(join)
		}

		if this.useCBO {
			joinOp := join.(plan.Operator)
			joinCost = joinOp.Cost()
			joinCard = joinOp.Cardinality()
		}
	}

	// there should only be a single match for each source document,
	// otherwise MERGE will return an error on multiple update/delete
	if this.useCBO && leftCard > 0.0 && joinCard > 0.0 && joinCard > leftCard {
		joinCard = leftCard
	}

	matchCard := OPT_CARD_NOT_AVAIL
	nonMatchCard := OPT_CARD_NOT_AVAIL
	if this.useCBO && leftCard > 0.0 && joinCard > 0.0 {
		matchCard = joinCard
		nonMatchCard = leftCard - joinCard
		if nonMatchCard < 1.0 {
			// assume at least 1 insert
			nonMatchCard = 1.0
			matchCard = leftCard - nonMatchCard
		}
	}

	actions := stmt.Actions()
	var update, delete, insert plan.Operator
	var updateCost, deleteCost, insertCost float64
	var updateCard, deleteCard, insertCard float64

	if actions.Update() != nil {
		act := actions.Update()
		ops := make([]plan.Operator, 0, 5)

		cost = OPT_COST_NOT_AVAIL
		if this.useCBO && joinCost > 0.0 {
			// do not use cumulative cost for embedded operators
			cost = optMinCost()
		}
		cardinality = matchCard
		if act.Where() != nil {
			filter := this.addMergeFilter(act.Where(), cost, cardinality)
			ops = append(ops, filter)
			cost = filter.Cost()
			cardinality = filter.Cardinality()
		}

		if this.useCBO && cost > 0.0 && cardinality > 0.0 {
			cost, cardinality = getCloneCost(keyspace, cost, cardinality)
		}
		ops = append(ops, plan.NewClone(ksref.Alias(), cost, cardinality))

		if act.Set() != nil {
			if this.useCBO && cost > 0.0 && cardinality > 0.0 {
				cost, cardinality = getUpdateSetCost(keyspace, act.Set(), cost, cardinality)
			}
			ops = append(ops, plan.NewSet(act.Set(), cost, cardinality))
		}

		if act.Unset() != nil {
			if this.useCBO && cost > 0.0 && cardinality > 0.0 {
				cost, cardinality = getUpdateUnsetCost(keyspace, act.Unset(), cost, cardinality)
			}
			ops = append(ops, plan.NewUnset(act.Unset(), cost, cardinality))
		}

		if this.useCBO && cost > 0.0 && cardinality > 0.0 {
			cost, cardinality = getUpdateSendCost(keyspace, stmt.Limit(), cost, cardinality)
		}
		ops = append(ops, plan.NewSendUpdate(keyspace, ksref, stmt.Limit(), cost, cardinality))
		update = plan.NewSequence(ops...)
		if this.useCBO && cost > 0.0 {
			updateCost = cost
			updateCard = cardinality
		}
	}

	if actions.Delete() != nil {
		act := actions.Delete()
		ops := make([]plan.Operator, 0, 4)

		cost = OPT_COST_NOT_AVAIL
		if this.useCBO && joinCost > 0.0 {
			// do not use cumulative cost for embedded operators
			cost = optMinCost()
		}
		cardinality = matchCard
		if act.Where() != nil {
			filter := this.addMergeFilter(act.Where(), cost, cardinality)
			ops = append(ops, filter)
			cost = filter.Cost()
			cardinality = filter.Cardinality()
		}

		if this.useCBO && cost > 0.0 && cardinality > 0.0 {
			cost, cardinality = getDeleteCost(keyspace, stmt.Limit(), cost, cardinality)
		}

		ops = append(ops, plan.NewSendDelete(keyspace, ksref, stmt.Limit(), cost, cardinality))
		delete = plan.NewSequence(ops...)
		if this.useCBO && cost > 0.0 {
			deleteCost = cost
			deleteCard = cardinality
		}
	}

	if actions.Insert() != nil {
		act := actions.Insert()
		ops := make([]plan.Operator, 0, 4)

		cost = OPT_COST_NOT_AVAIL
		if this.useCBO && joinCost > 0.0 {
			// do not use cumulative cost for embedded operators
			cost = optMinCost()
		}
		cardinality = nonMatchCard
		if act.Where() != nil {
			filter := this.addMergeFilter(act.Where(), cost, cardinality)
			ops = append(ops, filter)
			cost = filter.Cost()
			cardinality = filter.Cardinality()
		}

		var keyExpr expression.Expression
		if stmt.IsOnKey() {
			keyExpr = stmt.On()
		} else {
			keyExpr = act.Key()
		}

		if this.useCBO && cost > 0.0 && cardinality > 0.0 {
			cost, cardinality = getInsertCost(keyspace, keyExpr, act.Value(),
				act.Options(), stmt.Limit(), cost, cardinality)
		}

		ops = append(ops, plan.NewSendInsert(keyspace, ksref, keyExpr, act.Value(),
			act.Options(), stmt.Limit(), cost, cardinality))
		insert = plan.NewSequence(ops...)
		if this.useCBO && cost > 0.0 {
			insertCost = cost
			insertCard = cardinality
		}
	}

	if this.useCBO && joinCost > 0.0 && joinCard > 0.0 {
		cost = joinCost
		cardinality = 0.0
		if actions.Update() != nil {
			cost += updateCost
			cardinality += updateCard
		}
		if actions.Delete() != nil {
			cost += deleteCost
			cardinality += deleteCard
		}
		if actions.Insert() != nil {
			cost += insertCost
			cardinality += insertCard
		}
	}

	var mergeKey expression.Expression
	if stmt.IsOnKey() {
		mergeKey = stmt.On()
	}
	merge := plan.NewMerge(keyspace, ksref, mergeKey, update, delete, insert, cost, cardinality)
	this.addSubChildren(merge)

	if stmt.Returning() != nil {
		this.subChildren = this.buildDMLProject(stmt.Returning(), this.subChildren)
	}

	this.addChildren(this.addSubchildrenParallel())

	if stmt.Limit() != nil {
		this.addChildren(plan.NewLimit(stmt.Limit(), OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL))
	}

	if stmt.Returning() == nil {
		this.addChildren(plan.NewDiscard(cost, cardinality))
	}

	return plan.NewSequence(this.children...), nil
}

func (this *builder) addMergeFilter(pred expression.Expression, cost, cardinality float64) *plan.Filter {
	if this.useCBO {
		cost, cardinality = getFilterCostWithInput(pred, this.baseKeyspaces,
			this.keyspaceNames, cost, cardinality, this.advisorValidate())
	}

	return plan.NewFilter(pred, cost, cardinality)
}
