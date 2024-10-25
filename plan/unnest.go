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

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type Unnest struct {
	readonly
	optEstimate
	BuildBitFilterBase
	term   *algebra.Unnest
	alias  string
	filter expression.Expression
}

func NewUnnest(term *algebra.Unnest, filter expression.Expression, cost, cardinality float64,
	size int64, frCost float64) *Unnest {
	rv := &Unnest{
		term:   term,
		alias:  term.Alias(),
		filter: filter,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *Unnest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnnest(this)
}

func (this *Unnest) New() Operator {
	return &Unnest{}
}

func (this *Unnest) Term() *algebra.Unnest {
	return this.term
}

func (this *Unnest) Alias() string {
	return this.alias
}

func (this *Unnest) Filter() expression.Expression {
	return this.filter
}

func (this *Unnest) SetFilter(filter expression.Expression) {
	this.filter = filter
}

func (this *Unnest) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Unnest) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Unnest"}

	if this.term.Outer() {
		r["outer"] = this.term.Outer()
	}

	r["expr"] = this.term.Expression().String()
	if this.alias != "" {
		r["as"] = this.alias
	}

	if this.filter != nil {
		r["filter"] = this.filter.String()
	}

	if this.HasBuildBitFilter() {
		this.marshalBuildBitFilters(r)
	}

	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *Unnest) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_               string                 `json:"#operator"`
		Outer           bool                   `json:"outer"`
		Expr            string                 `json:"expr"`
		As              string                 `json:"as"`
		Filter          string                 `json:"filter"`
		OptEstimate     map[string]interface{} `json:"optimizer_estimates"`
		BuildBitFilters []json.RawMessage      `json:"build_bit_filters"`
	}
	var expr expression.Expression

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	if _unmarshalled.Expr != "" {
		expr, err = parser.Parse(_unmarshalled.Expr)
		if err != nil {
			return err
		}
	}

	if _unmarshalled.Filter != "" {
		this.filter, err = parser.Parse(_unmarshalled.Filter)
		if err != nil {
			return err
		}
	}

	if len(_unmarshalled.BuildBitFilters) > 0 {
		err = this.unmarshalBuildBitFilters(_unmarshalled.BuildBitFilters)
		if err != nil {
			return err
		}
	}

	this.term = algebra.NewUnnest(nil, _unmarshalled.Outer, expr, _unmarshalled.As)
	this.alias = _unmarshalled.As

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

	return nil
}
