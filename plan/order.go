//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import (
	"encoding/json"
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/value"
)

const (
	_EARLY_ORDER = uint32(1) << iota
	_CAN_SPILL
	_CLIP_VALUES
)

type Order struct {
	readonly
	optEstimate
	terms                algebra.SortTerms
	partialSortTermCount int
	flags                uint32
	offset               *Offset
	limit                *Limit
}

const _FALLBACK_NUM = 64 * 1024

func NewOrder(order *algebra.Order, partialSortTermCount int, offset *Offset, limit *Limit, cost, cardinality float64,
	size int64, frCost float64, clipValues bool, canSpill bool) *Order {
	rv := &Order{
		terms:                order.Terms(),
		partialSortTermCount: partialSortTermCount,
		offset:               offset,
		limit:                limit,
	}
	if clipValues {
		rv.flags |= _CLIP_VALUES
	}
	if canSpill {
		rv.flags |= _CAN_SPILL
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *Order) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitOrder(this)
}

func (this *Order) New() Operator {
	return &Order{}
}

func (this *Order) Terms() algebra.SortTerms {
	return this.terms
}

func (this *Order) PartialSortTermCount() int {
	return this.partialSortTermCount
}

func (this *Order) IsEarlyOrder() bool {
	return (this.flags & _EARLY_ORDER) != 0
}

func (this *Order) SetEarlyOrder() {
	this.flags |= _EARLY_ORDER
}

func (this *Order) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Order) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Order"}

	/* generate sort terms */
	s := make([]interface{}, 0, len(this.terms))
	for _, term := range this.terms {
		q := make(map[string]interface{})
		q["expr"] = term.Expression().String()
		if term.DescendingExpr() != nil {
			q["desc"] = term.DescendingExpr().String()
		}
		if term.NullsPosExpr() != nil {
			q["nulls_pos"] = term.NullsPosExpr().String()
		}

		s = append(s, q)
	}
	r["sort_terms"] = s
	if this.partialSortTermCount > 0 {
		r["partial_sort_term_count"] = this.partialSortTermCount
	}
	if this.flags > 0 {
		r["flags"] = this.flags
	}
	if this.offset != nil {
		r["offset"] = this.offset.Expression().String()
	}
	if this.limit != nil {
		r["limit"] = this.limit.Expression().String()
	}
	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}
	if f != nil {
		f(r)
	}
	return r
}

func (this *Order) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string `json:"#operator"`
		Terms []struct {
			Expr     string      `json:"expr"`
			Desc     interface{} `json:"desc"`
			NullsPos interface{} `json:"nulls_pos"`
		} `json:"sort_terms"`
		PartialSortTermCount int                    `json:"partial_sort_term_count"`
		Flags                uint32                 `json:"flags"`
		OffsetExpr           string                 `json:"offset"`
		LimitExpr            string                 `json:"limit"`
		OptEstimate          map[string]interface{} `json:"optimizer_estimates"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.terms = make(algebra.SortTerms, len(_unmarshalled.Terms))
	for i, term := range _unmarshalled.Terms {
		expr, err := parser.Parse(term.Expr)
		if err != nil {
			return err
		}
		var desc, nullsPos expression.Expression

		oldStylePermitted := true
		newStylePermitted := true
		switch tDesc := term.Desc.(type) {
		case nil:
		case bool:
			newStylePermitted = false
			if tDesc {
				desc = expression.NewConstant(value.NewValue("desc"))
			}
		case string:
			oldStylePermitted = false
			if tDesc != "" {
				desc, err = parser.Parse(tDesc)
				if err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("Invalid marshalled Order")

		}

		switch tNullsPos := term.NullsPos.(type) {
		case nil:
		case bool:
			if !oldStylePermitted {
				return fmt.Errorf("Invalid marshalled Order")
			}
			if bDesc, _ := term.Desc.(bool); bDesc {
				if tNullsPos == true {
					nullsPos = expression.NewConstant(value.NewValue("first"))
				}
			} else {
				if tNullsPos == true {
					nullsPos = expression.NewConstant(value.NewValue("last"))
				}
			}
		case string:
			if !newStylePermitted {
				return fmt.Errorf("Invalid marshalled Order")
			}
			if tNullsPos != "" {
				nullsPos, err = parser.Parse(tNullsPos)
				if err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("Invalid marshalled Order")
		}

		this.terms[i] = algebra.NewSortTerm(expr, desc, nullsPos)
	}
	this.flags = _unmarshalled.Flags
	if offsetExprStr := _unmarshalled.OffsetExpr; offsetExprStr != "" {
		offsetExpr, err := parser.Parse(offsetExprStr)
		if err != nil {
			return err
		}
		this.offset = NewOffset(offsetExpr, PLAN_COST_NOT_AVAIL, PLAN_CARD_NOT_AVAIL, PLAN_SIZE_NOT_AVAIL, PLAN_COST_NOT_AVAIL)
	}
	if limitExprStr := _unmarshalled.LimitExpr; limitExprStr != "" {
		limitExpr, err := parser.Parse(limitExprStr)
		if err != nil {
			return err
		}
		this.limit = NewLimit(limitExpr, PLAN_COST_NOT_AVAIL, PLAN_CARD_NOT_AVAIL, PLAN_SIZE_NOT_AVAIL, PLAN_COST_NOT_AVAIL)
	}

	this.partialSortTermCount = _unmarshalled.PartialSortTermCount
	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

	return nil
}

func (this *Order) LimitPushed() bool {
	return this.limit != nil
}

func (this *Order) Offset() *Offset {
	return this.offset
}

func (this *Order) Limit() *Limit {
	return this.limit
}

func (this *Order) ClipValues() bool {
	return (this.flags & _CLIP_VALUES) != 0
}

func (this *Order) CanSpill() bool {
	return (this.flags & _CAN_SPILL) != 0
}

func OrderFallbackNum() int {
	return _FALLBACK_NUM
}
