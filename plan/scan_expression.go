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

type ExpressionScan struct {
	readonly
	optEstimate
	BuildBitFilterBase
	fromExpr    expression.Expression
	alias       string
	correlated  bool
	nested_loop bool
	filter      expression.Expression
	subqPlan    Operator
}

func NewExpressionScan(fromExpr expression.Expression, alias string, correlated, nested_loop bool,
	filter expression.Expression, cost, cardinality float64, size int64, frCost float64) *ExpressionScan {
	rv := &ExpressionScan{
		fromExpr:    fromExpr,
		alias:       alias,
		correlated:  correlated,
		nested_loop: nested_loop,
		filter:      filter,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *ExpressionScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExpressionScan(this)
}

func (this *ExpressionScan) New() Operator {
	return &ExpressionScan{}
}

func (this *ExpressionScan) FromExpr() expression.Expression {
	return this.fromExpr
}

func (this *ExpressionScan) SetFromExpr(fromExpr expression.Expression) {
	this.fromExpr = fromExpr
}

func (this *ExpressionScan) Alias() string {
	return this.alias
}

func (this *ExpressionScan) IsCorrelated() bool {
	return this.correlated
}

func (this *ExpressionScan) IsUnderNL() bool {
	return this.nested_loop
}

func (this *ExpressionScan) Filter() expression.Expression {
	return this.filter
}

func (this *ExpressionScan) SetFilter(filter expression.Expression) {
	this.filter = filter
}

// subqPlan: in case a SubqueryTerm is used under inner of a nested-loop join, we put an
// ExpressionScan on top of the subquery; in this case we need to add the query plan of
// the subquery in the "~subqueries" section of explain plan.

func (this *ExpressionScan) SubqueryPlan() Operator {
	return this.subqPlan
}

func (this *ExpressionScan) SetSubqueryPlan(subqPlan Operator) {
	this.subqPlan = subqPlan
}

func (this *ExpressionScan) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *ExpressionScan) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "ExpressionScan"}
	r["expr"] = this.fromExpr.String()
	r["alias"] = this.alias
	if !this.correlated {
		r["uncorrelated"] = !this.correlated
	}
	if this.nested_loop {
		r["nested_loop"] = this.nested_loop
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

func (this *ExpressionScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_               string                 `json:"#operator"`
		FromExpr        string                 `json:"expr"`
		Alias           string                 `json:"alias"`
		UnCorrelated    bool                   `json:"uncorrelated"`
		NestedLoop      bool                   `json:"nested_loop"`
		Filter          string                 `json:"filter"`
		OptEstimate     map[string]interface{} `json:"optimizer_estimates"`
		BuildBitFilters []json.RawMessage      `json:"build_bit_filters"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	if _unmarshalled.FromExpr != "" {
		this.fromExpr, err = parser.Parse(_unmarshalled.FromExpr)
	}
	this.alias = _unmarshalled.Alias
	// we use uncorrelated in marshall such that in mixed node cluster
	// where a query node can be running an earlier version of N1QL
	// and thus generate plan without the correlated information,
	// we set correlated to be true just to be safe, i.e., if
	// no info in the plan, then assume correlated is true.
	this.correlated = !_unmarshalled.UnCorrelated
	this.nested_loop = _unmarshalled.NestedLoop

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

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

	planContext := this.PlanContext()
	if planContext != nil {
		_, err = planContext.Map(this.fromExpr)
		if err != nil {
			return err
		}
		planContext.addExprTermAlias(this.alias)
		if this.filter != nil {
			_, err = planContext.Map(this.filter)
			if err != nil {
				return err
			}
		}
	}

	return err
}
