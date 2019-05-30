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
	fromExpr    expression.Expression
	alias       string
	correlated  bool
	cost        float64
	cardinality float64
}

func NewExpressionScan(fromExpr expression.Expression, alias string, correlated bool, cost, cardinality float64) *ExpressionScan {
	return &ExpressionScan{
		fromExpr:    fromExpr,
		alias:       alias,
		correlated:  correlated,
		cost:        cost,
		cardinality: cardinality,
	}
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

func (this *ExpressionScan) Cost() float64 {
	return this.cost
}

func (this *ExpressionScan) Cardinality() float64 {
	return this.cardinality
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
	if this.cost > 0.0 {
		r["cost"] = this.cost
	}
	if this.cardinality > 0.0 {
		r["cardinality"] = this.cardinality
	}
	if f != nil {
		f(r)
	}
	return r
}

func (this *ExpressionScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_            string  `json:"#operator"`
		FromExpr     string  `json:"expr"`
		Alias        string  `json:"alias"`
		UnCorrelated bool    `json:"uncorrelated"`
		Cost         float64 `json:"cost"`
		Cardinality  float64 `json:"cardinality"`
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

	if _unmarshalled.Cost > 0.0 {
		this.cost = _unmarshalled.Cost
	} else {
		this.cost = PLAN_COST_NOT_AVAIL
	}

	if _unmarshalled.Cardinality > 0.0 {
		this.cardinality = _unmarshalled.Cardinality
	} else {
		this.cardinality = PLAN_CARD_NOT_AVAIL
	}

	return err
}
