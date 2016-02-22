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
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
)

// SELECT

func (this *builder) VisitSelect(stmt *algebra.Select) (interface{}, error) {
	// Restore previous values when exiting. VisitSelect()
	// can be called multiple times by set operators
	prevCover := this.cover
	prevOrder := this.order
	prevLimit := this.limit
	prevProjection := this.delayProjection
	defer func() {
		this.cover = prevCover
		this.order = prevOrder
		this.limit = prevLimit
		this.delayProjection = prevProjection
	}()

	order := stmt.Order()
	offset := stmt.Offset()
	limit := stmt.Limit()

	this.order = order
	if order != nil {
		// If there is an ORDER BY, delay the final projection
		this.delayProjection = true
		this.cover = stmt
	} else {
		this.cover = nil
	}

	if limit != nil {
		if offset != nil {
			this.limit = expression.NewAdd(offset, limit)
		} else {
			this.limit = limit
		}
	}

	sub, err := stmt.Subresult().Accept(this)
	if err != nil {
		return nil, err
	}

	if order == nil && offset == nil && limit == nil {
		return sub, nil
	}

	children := make([]plan.Operator, 0, 5)
	children = append(children, sub.(plan.Operator))

	if order != nil && this.order == nil {
		if limit != nil {
			if offset != nil {
				children = append(children, plan.NewOrder(order, plan.NewOffset(offset), plan.NewLimit(limit)))
			} else {
				children = append(children, plan.NewOrder(order, nil, plan.NewLimit(limit)))
			}
		} else {
			children = append(children, plan.NewOrder(order, nil, nil))
		}
	}

	if offset != nil {
		children = append(children, plan.NewOffset(offset))
	}

	if limit != nil {
		children = append(children, plan.NewLimit(limit))
	}

	// Perform the delayed final projection now, after the ORDER BY
	if this.delayProjection {
		children = append(children, plan.NewFinalProject())
	}

	return plan.NewSequence(children...), nil
}
