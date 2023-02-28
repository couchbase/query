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
	sourceKeyspace, duration := base.NewBaseKeyspace(source.Alias(), path, nil, 1)
	this.recordSubTime("keyspace.metadata", duration)
	this.baseKeyspaces[sourceKeyspace.Name()] = sourceKeyspace
	targetKeyspace, duration := base.NewBaseKeyspace(stmt.KeyspaceRef().Alias(), stmt.KeyspaceRef().Path(), nil, 2)
	this.recordSubTime("keyspace.metadata", duration)
	this.baseKeyspaces[targetKeyspace.Name()] = targetKeyspace
	this.collectKeyspaceNames()

	this.skipKeyspace = targetKeyspace.Keyspace()

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
		left = source.SubqueryTerm()
	} else if source.ExpressionTerm() != nil {
		left = source.ExpressionTerm()
	} else {
		if source.From() == nil {
			// should have caught in semantics check
			return nil, errors.NewPlanInternalError("VisitMerge: MERGE missing source.")
		}
		left = source.From()
	}
	sourceKeyspace.SetNode(left)

	ksref := stmt.KeyspaceRef()
	ksref.SetDefaultNamespace(this.namespace)
	ksAlias := ksref.Alias()

	right := algebra.NewKeyspaceTermFromPath(ksref.Path(), ksref.As(), nil, stmt.Indexes())
	targetKeyspace.SetNode(right)

	qp := plan.NewQueryPlan(nil)
	err = this.chkBldSubqueries(stmt, qp)
	if err != nil {
		return nil, err
	}

	stmt.SetOptimHints(deriveOptimHints(this.baseKeyspaces, stmt.OptimHints()))
	optimHints := stmt.OptimHints()
	if optimHints != nil {
		this.processOptimHints(optimHints)
		markDMLOrderedHintError(optimHints)
	}

	if stmt.IsOnKey() {
		targetKeyspace.MarkJoinHintError(algebra.MERGE_ONKEY_JOIN_HINT_ERR)
		targetKeyspace.MarkIndexHintError(algebra.MERGE_ONKEY_INDEX_HINT_ERR)
		sourceKeyspace.MarkJoinHintError(algebra.MERGE_ONKEY_JOIN_HINT_ERR)

	} else {
		leftJoinHint := sourceKeyspace.JoinHint()
		rightJoinHint := targetKeyspace.JoinHint()
		if leftJoinHint != algebra.JOIN_HINT_NONE && rightJoinHint != algebra.JOIN_HINT_NONE {
			sourceKeyspace.SetJoinHintError()
			targetKeyspace.SetJoinHintError()
		} else if leftJoinHint != algebra.JOIN_HINT_NONE {
			right.SetInferJoinHint()
			left.SetTransferJoinHint()
		}
	}

	_, err = left.Accept(this)
	if err != nil && !this.indexAdvisor {
		return nil, err
	}

	keyspace, err := this.getNameKeyspace(ksref, false)
	if err != nil {
		return nil, err
	}

	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL
	size := OPT_SIZE_NOT_AVAIL
	frCost := OPT_COST_NOT_AVAIL
	joinCost := OPT_COST_NOT_AVAIL
	joinCard := OPT_CARD_NOT_AVAIL
	joinFrCost := OPT_COST_NOT_AVAIL
	leftCard := OPT_CARD_NOT_AVAIL

	if this.useCBO && this.lastOp != nil {
		leftCard = this.lastOp.Cardinality()
	}

	if stmt.IsOnKey() {
		if this.useCBO && this.keyspaceUseCBO(ksAlias) {
			rightKeyspace := base.GetKeyspaceName(this.baseKeyspaces, ksAlias)
			joinCost, joinCard, _, joinFrCost = getLookupJoinCost(this.lastOp, outer, right,
				rightKeyspace)
		}
	} else {
		// use ANSI JOIN to handle the ON-clause
		right.SetAnsiJoin()

		ansiJoin := algebra.NewAnsiJoin(left, outer, right, stmt.On())
		ansiJoin.SetPushable(outer == false)
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
			joinFrCost = joinOp.FrCost()
		}
	}

	// there should only be a single match for each source document,
	// otherwise MERGE will return an error on multiple update/delete
	if this.useCBO && leftCard > 0.0 && joinCard > 0.0 && joinCard > leftCard {
		joinCard = leftCard
	}

	matchCard := OPT_CARD_NOT_AVAIL
	nonMatchCard := OPT_CARD_NOT_AVAIL
	targetSize := OPT_SIZE_NOT_AVAIL
	if this.useCBO && leftCard > 0.0 && joinCard > 0.0 {
		matchCard = joinCard
		nonMatchCard = leftCard - joinCard
		if nonMatchCard < 1.0 {
			// assume at least 1 insert
			nonMatchCard = 1.0
			matchCard = leftCard - nonMatchCard
		}

		targetSize = getKeyspaceSize(targetKeyspace.Keyspace())
	}

	actions := stmt.Actions()
	var update, delete, insert plan.Operator
	var updateFilter, deleteFilter, insertFilter expression.Expression
	var updateCost, deleteCost, insertCost float64
	var updateCard, deleteCard, insertCard float64
	var updateFrCost, deleteFrCost, insertFrCost float64

	if actions.Update() != nil {
		act := actions.Update()
		ops := make([]plan.Operator, 0, 4)

		cost = OPT_COST_NOT_AVAIL
		frCost = OPT_COST_NOT_AVAIL
		size = OPT_SIZE_NOT_AVAIL
		if this.useCBO && joinCost > 0.0 {
			// do not use cumulative cost for embedded operators
			cost = optMinCost()
			frCost = optMinCost()
			size = targetSize
		}
		cardinality = matchCard
		updateFilter, err = expression.RemoveConstants(act.Where())
		if err != nil {
			return nil, err
		}
		if updateFilter != nil {
			cost, cardinality, size, frCost = this.addMergeFilterCost(updateFilter, ksAlias, cost, cardinality, size, frCost)
		}

		if this.useCBO && cost > 0.0 && cardinality > 0.0 && size > 0 && frCost > 0.0 {
			cost, cardinality, size, frCost = getCloneCost(cost, cardinality, size, frCost)
		}
		ops = append(ops, plan.NewClone(ksAlias, cost, cardinality, size, frCost))

		if act.Set() != nil {
			if this.useCBO && cost > 0.0 && cardinality > 0.0 && size > 0 && frCost > 0.0 {
				cost, cardinality, size, frCost = getUpdateSetCost(act.Set(),
					cost, cardinality, size, frCost)
			}
			ops = append(ops, plan.NewSet(act.Set(), cost, cardinality, size, frCost))
		}

		if act.Unset() != nil {
			if this.useCBO && cost > 0.0 && cardinality > 0.0 && size > 0 && frCost > 0.0 {
				cost, cardinality, size, frCost = getUpdateUnsetCost(act.Unset(),
					cost, cardinality, size, frCost)
			}
			ops = append(ops, plan.NewUnset(act.Unset(), cost, cardinality, size, frCost))
		}

		if this.useCBO && cost > 0.0 && cardinality > 0.0 && size > 0 && frCost > 0.0 {
			cost, cardinality, size, frCost = getUpdateSendCost(stmt.Limit(),
				cost, cardinality, size, frCost)
		}
		ops = append(ops, plan.NewSendUpdate(keyspace, ksref, stmt.Limit(), cost, cardinality, size, frCost, stmt.Returning() == nil))
		update = plan.NewSequence(ops...)
		if this.useCBO && cost > 0.0 {
			updateCost = cost
			updateCard = cardinality
			updateFrCost = frCost
		}
	}

	if actions.Delete() != nil {
		act := actions.Delete()

		cost = OPT_COST_NOT_AVAIL
		frCost = OPT_COST_NOT_AVAIL
		size = OPT_SIZE_NOT_AVAIL
		if this.useCBO && joinCost > 0.0 {
			// do not use cumulative cost for embedded operators
			cost = optMinCost()
			frCost = optMinCost()
			size = targetSize
		}
		cardinality = matchCard
		deleteFilter, err = expression.RemoveConstants(act.Where())
		if err != nil {
			return nil, err
		}
		if deleteFilter != nil {
			cost, cardinality, size, frCost = this.addMergeFilterCost(deleteFilter, ksAlias, cost, cardinality, size, frCost)
		}

		if this.useCBO && cost > 0.0 && cardinality > 0.0 && size > 0 && frCost > 0.0 {
			cost, cardinality, size, frCost = getDeleteCost(stmt.Limit(),
				cost, cardinality, size, frCost)
		}

		delete = plan.NewSendDelete(keyspace, ksref, stmt.Limit(), cost, cardinality, size, frCost, stmt.Returning() == nil)
		if this.useCBO && cost > 0.0 {
			deleteCost = cost
			deleteCard = cardinality
			deleteFrCost = frCost
		}
	}

	if actions.Insert() != nil {
		act := actions.Insert()

		cost = OPT_COST_NOT_AVAIL
		frCost = OPT_COST_NOT_AVAIL
		size = OPT_SIZE_NOT_AVAIL
		if this.useCBO && joinCost > 0.0 {
			// do not use cumulative cost for embedded operators
			cost = optMinCost()
			frCost = optMinCost()
			size = targetSize
		}
		cardinality = nonMatchCard
		insertFilter, err = expression.RemoveConstants(act.Where())
		if err != nil {
			return nil, err
		}
		if insertFilter != nil {
			cost, cardinality, size, frCost = this.addMergeFilterCost(insertFilter, ksAlias, cost, cardinality, size, frCost)
		}

		var keyExpr expression.Expression
		if stmt.IsOnKey() {
			keyExpr = stmt.On()
		} else {
			keyExpr = act.Key()
		}

		if this.useCBO && cost > 0.0 && cardinality > 0.0 && size > 0 && frCost > 0.0 {
			cost, cardinality, size, frCost = getInsertCost(keyExpr, act.Value(),
				act.Options(), stmt.Limit(), cost, cardinality, size, frCost)
		}

		insert = plan.NewSendInsert(keyspace, ksref, keyExpr, act.Value(), act.Options(), stmt.Limit(), cost, cardinality, size,
			frCost, this.mustSkipKeys, stmt.Returning() == nil)
		if this.useCBO && cost > 0.0 {
			insertCost = cost
			insertCard = cardinality
			insertFrCost = frCost
		}
	}

	if this.useCBO && joinCost > 0.0 && joinCard > 0.0 && targetSize > 0 && joinFrCost > 0.0 {
		cost = joinCost
		frCost = joinFrCost
		cardinality = 0.0
		size = targetSize
		if actions.Update() != nil {
			cost += updateCost
			cardinality += updateCard
			frCost += updateFrCost
		}
		if actions.Delete() != nil {
			cost += deleteCost
			cardinality += deleteCard
			frCost += deleteFrCost
		}
		if actions.Insert() != nil {
			cost += insertCost
			cardinality += insertCard
			frCost += insertFrCost
		}
	}

	var mergeKey expression.Expression
	if stmt.IsOnKey() {
		mergeKey = stmt.On()
	}
	merge := plan.NewMerge(keyspace, ksref, mergeKey, update, updateFilter, delete, deleteFilter, insert, insertFilter, cost,
		cardinality, size, frCost)
	this.addSubChildren(merge)

	if stmt.Returning() != nil {
		this.subChildren = this.buildDMLProject(stmt.Returning(), this.subChildren)
	}

	this.addChildren(this.addSubchildrenParallel())

	if stmt.Limit() != nil {
		if this.useCBO && cost > 0.0 && cardinality > 0.0 && size > 0 && frCost > 0.0 {
			nlimit := int64(0)
			lv, static := base.GetStaticInt(stmt.Limit())
			if static {
				nlimit = lv
			}
			cost, cardinality, size, frCost = getLimitCost(this.lastOp, nlimit, -1)
		}
		this.addChildren(plan.NewLimit(stmt.Limit(), cost, cardinality, size, frCost))
	}

	qp.SetPlanOp(plan.NewSequence(this.children...))
	return qp, nil
}

func (this *builder) addMergeFilterCost(pred expression.Expression, alias string,
	cost, cardinality float64, size int64, frCost float64) (float64, float64, int64, float64) {
	if this.useCBO {
		cost, cardinality, size, frCost = getFilterCostWithInput(pred, this.baseKeyspaces,
			this.keyspaceNames, alias, cost, cardinality, size, frCost,
			this.advisorValidate(), this.context)
	}
	return cost, cardinality, size, frCost
}
