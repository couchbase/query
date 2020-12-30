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

	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type ExpressionScan struct {
	readonly
	optEstimate
	fromExpr   expression.Expression
	alias      string
	correlated bool
	filter     expression.Expression
}

func NewExpressionScan(fromExpr expression.Expression, alias string, correlated bool,
	filter expression.Expression, cost, cardinality float64) *ExpressionScan {
	rv := &ExpressionScan{
		fromExpr:   fromExpr,
		alias:      alias,
		correlated: correlated,
		filter:     filter,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality)
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

func (this *ExpressionScan) Alias() string {
	return this.alias
}

func (this *ExpressionScan) IsCorrelated() bool {
	return this.correlated
}

func (this *ExpressionScan) Filter() expression.Expression {
	return this.filter
}

func (this *ExpressionScan) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *ExpressionScan) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "ExpressionScan"}
	r["expr"] = expression.NewStringer().Visit(this.fromExpr)
	r["alias"] = this.alias
	if !this.correlated {
		r["uncorrelated"] = !this.correlated
	}
	if this.filter != nil {
		r["filter"] = expression.NewStringer().Visit(this.filter)
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
		_            string                 `json:"#operator"`
		FromExpr     string                 `json:"expr"`
		Alias        string                 `json:"alias"`
		UnCorrelated bool                   `json:"uncorrelated"`
		Filter       string                 `json:"filter"`
		OptEstimate  map[string]interface{} `json:"optimizer_estimates"`
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

	if _unmarshalled.Filter != "" {
		this.filter, err = parser.Parse(_unmarshalled.Filter)
		if err != nil {
			return err
		}
	}

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

	return err
}
