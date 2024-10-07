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

	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type Filter struct {
	readonly
	optEstimate
	BuildBitFilterBase
	cond  expression.Expression
	alias string
}

func NewFilter(cond expression.Expression, alias string, cost, cardinality float64, size int64, frCost float64) *Filter {
	rv := &Filter{
		cond:  cond,
		alias: alias,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *Filter) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFilter(this)
}

func (this *Filter) New() Operator {
	return &Filter{}
}

func (this *Filter) Condition() expression.Expression {
	return this.cond
}

func (this *Filter) SetCondition(cond expression.Expression) {
	this.cond = cond
}

func (this *Filter) Alias() string {
	return this.alias
}

func (this *Filter) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Filter) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Filter"}
	r["condition"] = expression.NewStringer().Visit(this.cond)
	if this.alias != "" {
		r["alias"] = this.alias
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

func (this *Filter) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_               string                 `json:"#operator"`
		Condition       string                 `json:"condition"`
		Alias           string                 `json:"alias"`
		OptEstimate     map[string]interface{} `json:"optimizer_estimates"`
		BuildBitFilters []json.RawMessage      `json:"build_bit_filters"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	if _unmarshalled.Condition != "" {
		this.cond, err = parser.Parse(_unmarshalled.Condition)
		if err != nil {
			return err
		}
	}

	if _unmarshalled.Alias != "" {
		this.alias = _unmarshalled.Alias
	}

	if len(_unmarshalled.BuildBitFilters) > 0 {
		err = this.unmarshalBuildBitFilters(_unmarshalled.BuildBitFilters)
		if err != nil {
			return err
		}
	}

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

	return nil
}
