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
	defer func() {
		this.cover = prevCover
		this.order = prevOrder
		this.limit = prevLimit
		this.offset = prevOffset
		this.delayProjection = prevProjection
	}()

	stmtOrder := stmt.Order()
	stmtOffset, err := newExpr(stmt.Offset(), true)
	if err != nil {
		return nil, err
	}

	stmtLimit, err := newExpr(stmt.Limit(), false)
	if err != nil {
		return nil, err
	}

	this.cover = nil
	this.delayProjection = false
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

	if stmtOrder != nil && this.order == nil {
		if stmtLimit != nil {
			if stmtOffset != nil && this.offset == nil {
				children = append(children, plan.NewOrder(stmtOrder, plan.NewOffset(stmtOffset), plan.NewLimit(stmtLimit)))
			} else {
				children = append(children, plan.NewOrder(stmtOrder, nil, plan.NewLimit(stmtLimit)))
			}
		} else {
			children = append(children, plan.NewOrder(stmtOrder, nil, nil))
		}
	}

	if stmtOffset != nil && this.offset == nil {
		children = append(children, plan.NewOffset(stmtOffset))
	}

	if stmtLimit != nil {
		children = append(children, plan.NewLimit(stmtLimit))
	}

	// Perform the delayed final projection now, after the ORDER BY
	if this.delayProjection {
		children = append(children, plan.NewFinalProject())
	}

	return plan.NewSequence(children...), nil
}

func newExpr(expr expression.Expression, offset bool) (expression.Expression, error) {
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
