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
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/value"
)

// SELECT

func (this *builder) VisitSelect(stmt *algebra.Select) (interface{}, error) {
	// Restore previous values when exiting. VisitSelect()
	// can be called multiple times by set operators
	prevCover := this.cover
	prevOrder := this.order
	prevLimit := this.limit
	prevOffset := this.offset
	prevProjection := this.delayProjection
	prevRequirePrimaryKey := this.requirePrimaryKey
	prevCollectQueryInfo := this.storeCollectQueryInfo()
	defer func() {
		this.cover = prevCover
		this.order = prevOrder
		this.limit = prevLimit
		this.offset = prevOffset
		this.delayProjection = prevProjection
		this.requirePrimaryKey = prevRequirePrimaryKey
		this.restoreCollectQueryInfo(prevCollectQueryInfo)

	}()

	stmtOrder := stmt.Order()
	stmtOffset, err := newOffsetLimitExpr(stmt.Offset(), true)
	if err != nil {
		return nil, err
	}

	stmtLimit, err := newOffsetLimitExpr(stmt.Limit(), false)
	if err != nil {
		return nil, err
	}

	this.initialIndexAdvisor(stmt)

	this.cover = nil
	this.delayProjection = false
	this.requirePrimaryKey = false
	this.offset = stmtOffset
	this.limit = stmtLimit
	this.order = stmtOrder

	this.extractPagination(this.order, this.offset, this.limit)

	if stmtOrder != nil {
		// If there is an ORDER BY, delay the final projection
		this.delayProjection = true
		this.cover = stmt
	}

	qp := plan.NewQueryPlan(nil)
	err = this.chkBldSubqueries(stmt, qp)
	if err != nil {
		return nil, err
	}

	sub, err := stmt.Subresult().Accept(this)
	if err != nil {
		return nil, err
	}
	subOp := sub.(plan.Operator)

	addFromSubqueries(qp, stmt.OptimHints(), subOp)

	with := stmt.With()
	if with != nil {
		cost := OPT_COST_NOT_AVAIL
		cardinality := OPT_CARD_NOT_AVAIL
		size := OPT_SIZE_NOT_AVAIL
		frCost := OPT_COST_NOT_AVAIL
		if this.useCBO {
			cost, cardinality, size, frCost = getWithCost(subOp, with)
		}
		subOp = plan.NewWith(with, subOp, cost, cardinality, size, frCost)
	}

	if stmtOrder == nil && stmtOffset == nil && stmtLimit == nil {
		qp.SetPlanOp(subOp)
		return qp, nil
	}

	children := make([]plan.Operator, 0, 5)
	children = append(children, subOp)
	lastOp := subOp
	cost := lastOp.Cost()
	cardinality := lastOp.Cardinality()
	size := lastOp.Size()
	frCost := lastOp.FrCost()
	nlimit := int64(-1)
	noffset := int64(-1)
	if this.useCBO && (cost > 0.0) && (cardinality > 0.0) {
		if stmtLimit != nil {
			lv, static := base.GetStaticInt(stmtLimit)
			if static {
				nlimit = lv
			} else {
				nlimit = 0
			}
		}
		if stmtOffset != nil {
			ov, static := base.GetStaticInt(stmtOffset)
			if static {
				noffset = ov
			} else {
				noffset = 0
			}
		}
	}

	if stmtOrder != nil && !this.hasBuilderFlag(BUILDER_PLAN_HAS_ORDER|BUILDER_HAS_EARLY_ORDER) &&
		this.order == nil {

		var limit *plan.Limit
		var offset *plan.Offset
		if stmtLimit != nil {
			// the limit/offset operator that's embedded inside sort operator does not need cost
			// since only the corresponding expression is saved in the plan
			limit = plan.NewLimit(stmtLimit, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL)
			if stmtOffset != nil && this.offset == nil {
				offset = plan.NewOffset(stmtOffset, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL)
			}
		}
		if this.useCBO && (cost > 0.0) && (cardinality > 0.0) && (size > 0) && (frCost > 0.0) {
			scost, scardinality, ssize, sfrCost := getSortCost(size, len(stmtOrder.Terms()),
				cardinality, nlimit, noffset)
			if scost > 0.0 && scardinality > 0.0 && ssize > 0 && sfrCost > 0.0 {
				cost += scost
				cardinality = scardinality
				size = ssize
				frCost += sfrCost
			} else {
				cost = OPT_COST_NOT_AVAIL
				cardinality = OPT_CARD_NOT_AVAIL
				size = OPT_SIZE_NOT_AVAIL
				frCost = OPT_COST_NOT_AVAIL
			}
		}
		order := plan.NewOrder(stmtOrder, offset, limit, cost, cardinality, size, frCost, true)
		children = append(children, order)
		lastOp = order
	}

	if stmtOffset != nil && this.offset == nil {
		if this.useCBO && (cost > 0.0) && (cardinality > 0.0) && (size > 0) && (frCost > 0.0) {
			cost, cardinality, size, frCost = getOffsetCost(lastOp, noffset)
		}
		offset := plan.NewOffset(stmtOffset, cost, cardinality, size, frCost)
		children = append(children, offset)
		lastOp = offset
	}

	if stmtLimit != nil {
		if this.useCBO && (cost > 0.0) && (cardinality > 0.0) && (size > 0) && (frCost > 0.0) {
			cost, cardinality, size, frCost = getLimitCost(lastOp, nlimit, noffset)
		}
		limit := plan.NewLimit(stmtLimit, cost, cardinality, size, frCost)
		children = append(children, limit)
		lastOp = limit
	}

	// Perform the delayed final projection now, after the ORDER BY
	if this.delayProjection {

		// TODO retire
		children = maybeFinalProject(children)
	}

	qp.SetPlanOp(plan.NewSequence(children...))
	return qp, nil
}

func newOffsetLimitExpr(expr expression.Expression, offset bool) (expression.Expression, error) {
	if expr == nil {
		return expr, nil
	}

	val := expr.Value()
	if val == nil || val.Type() <= value.NULL {
		return expr, nil
	}

	actual := val.ActualForIndex()
	switch actual := actual.(type) {
	case float64:
		if value.IsInt(actual) {
			if offset && int64(actual) <= 0 {
				return nil, nil
			} else if !offset && int64(actual) < 0 {
				return expression.NewConstant(0), nil
			}
			return expr, nil
		}
	case int64:
		if offset && int64(actual) <= 0 {
			return nil, nil
		} else if !offset && int64(actual) < 0 {
			return expression.NewConstant(0), nil
		}
		return expr, nil
	}

	if offset {
		return nil, errors.NewInvalidValueError(fmt.Sprintf("Invalid OFFSET value %v", actual))
	}
	return nil, errors.NewInvalidValueError(fmt.Sprintf("Invalid LIMIT value %v", actual))
}

// check for any SubqueryTerm that falls under inner of nested-loop join, in which case we build an
// ExpressionScan on top of the subquery; need to add the subquery and its plan to "~subqueries"
func addFromSubqueries(qp *plan.QueryPlan, optimHints *algebra.OptimHints, ops ...plan.Operator) {
	for _, op := range ops {
		switch op := op.(type) {
		case *plan.ExpressionScan:
			if subq, ok := op.FromExpr().(*algebra.Subquery); ok {
				o := op.SubqueryPlan()
				if o != nil {
					subSelect := subq.Select()
					qp.AddSubquery(subSelect, o)
					// optimizer hints from the SubqueryTerm was added to
					// the parent query's optimizer hints; since the SubqueryTerm
					// now appears in "~subqueries" section it'll have its own
					// optimizer hints, and thus no longer need to be included
					// in the parent query's optimizer hints
					removeSubqueryTermHints(optimHints, op.Alias())
					// nested SubqueryTerm?
					addFromSubqueries(qp, subSelect.OptimHints(), o)
				}
			}
		case *plan.Parallel:
			addFromSubqueries(qp, optimHints, op.Child())
		case *plan.Sequence:
			addFromSubqueries(qp, optimHints, op.Children()...)
		case *plan.NLJoin:
			addFromSubqueries(qp, optimHints, op.Child())
		case *plan.NLNest:
			addFromSubqueries(qp, optimHints, op.Child())
		case *plan.HashJoin:
			addFromSubqueries(qp, optimHints, op.Child())
		case *plan.HashNest:
			addFromSubqueries(qp, optimHints, op.Child())
		}
	}
}
