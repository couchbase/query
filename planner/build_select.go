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

	if stmtOrder != nil {
		// If there is an ORDER BY, delay the final projection
		this.delayProjection = true
		this.cover = stmt
	}

	sub, err := stmt.Subresult().Accept(this)
	if err != nil {
		return nil, err
	}

	if stmtOrder == nil && stmtOffset == nil && stmtLimit == nil {
		return sub, nil
	}

	children := make([]plan.Operator, 0, 5)
	children = append(children, sub.(plan.Operator))
	lastOp := sub.(plan.Operator)
	cost := lastOp.Cost()
	cardinality := lastOp.Cardinality()
	nlimit := int64(0)
	noffset := int64(0)
	projSize := stmt.Subresult().EstResultSize()
	if this.useCBO && (cost > 0.0) && (cardinality > 0.0) {
		if stmtLimit != nil {
			lv, static := base.GetStaticInt(stmtLimit)
			if static {
				nlimit = lv
			}
		}
		if stmtOffset != nil {
			ov, static := base.GetStaticInt(stmtOffset)
			if static {
				noffset = ov
			}
		}
	}

	if stmtOrder != nil && this.order == nil {
		var limit *plan.Limit
		var offset *plan.Offset
		if stmtLimit != nil {
			// the limit/offset operator that's embedded inside sort operator does not need cost
			// since only the corresponding expression is saved in the plan
			limit = plan.NewLimit(stmtLimit, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL)
			if stmtOffset != nil && this.offset == nil {
				offset = plan.NewOffset(stmtOffset, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL)
			}
		}
		if this.useCBO && (cost > 0.0) && (cardinality > 0.0) {
			scost, scardinality := getSortCostWithSize(projSize, len(stmtOrder.Terms()),
				cardinality, nlimit, noffset)
			if scost > 0.0 && scardinality > 0.0 {
				cost += scost
				cardinality = scardinality
			} else {
				cost = OPT_COST_NOT_AVAIL
				cardinality = OPT_CARD_NOT_AVAIL
			}
		}
		order := plan.NewOrder(stmtOrder, offset, limit, cost, cardinality)
		children = append(children, order)
		lastOp = order
	}

	if stmtOffset != nil && this.offset == nil {
		if this.useCBO && (cost > 0.0) && (cardinality > 0.0) {
			cost, cardinality = getOffsetCost(projSize, lastOp, noffset)
		}
		offset := plan.NewOffset(stmtOffset, cost, cardinality)
		children = append(children, offset)
		lastOp = offset
	}

	if stmtLimit != nil {
		if this.useCBO && (cost > 0.0) && (cardinality > 0.0) {
			cost, cardinality = getLimitCost(projSize, lastOp, nlimit)
		}
		limit := plan.NewLimit(stmtLimit, cost, cardinality)
		children = append(children, limit)
		lastOp = limit
	}

	// Perform the delayed final projection now, after the ORDER BY
	if this.delayProjection {

		// TODO retire
		children = maybeFinalProject(children)
	}

	return plan.NewSequence(children...), nil
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
