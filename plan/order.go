//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plan

import (
	"encoding/json"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/expression/parser"
)

type Order struct {
	readonly
	terms  algebra.SortTerms
	offset *Offset
	limit  *Limit
}

const _FALLBACK_NUM = 64 * 1024

func NewOrder(order *algebra.Order, offset *Offset, limit *Limit) *Order {
	return &Order{
		terms:  order.Terms(),
		offset: offset,
		limit:  limit,
	}
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

func (this *Order) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "Order"}

	/* generate sort terms */
	s := make([]interface{}, 0, len(this.terms))
	for _, term := range this.terms {
		q := make(map[string]interface{})
		q["expr"] = term.Expression().String()

		if term.Descending() {
			q["desc"] = term.Descending()
		}

		s = append(s, q)
	}
	r["sort_terms"] = s
	if this.offset != nil {
		r["offset"] = this.offset.Expression().String()
	}
	if this.limit != nil {
		r["limit"] = this.limit.Expression().String()
	}
	if this.duration != 0 {
		r["#time"] = this.duration.String()
	}
	return json.Marshal(r)
}

func (this *Order) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string `json:"#operator"`
		Terms []struct {
			Expr string `json:"expr"`
			Desc bool   `json:"desc"`
		} `json:"sort_terms"`
		offsetExpr string `json:"offset"`
		limitExpr  string `json:"limit"`
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
		this.terms[i] = algebra.NewSortTerm(expr, term.Desc)
	}
	if offsetExprStr := _unmarshalled.offsetExpr; offsetExprStr != "" {
		offsetExpr, err := parser.Parse(offsetExprStr)
		if err != nil {
			return err
		}
		this.offset = NewOffset(offsetExpr)
	}
	if limitExprStr := _unmarshalled.limitExpr; limitExprStr != "" {
		limitExpr, err := parser.Parse(limitExprStr)
		if err != nil {
			return err
		}
		this.limit = NewLimit(limitExpr)
	}
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

func (this *Order) FallbackNum() int {
	return _FALLBACK_NUM
}
