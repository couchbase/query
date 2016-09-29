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
	fromExpr expression.Expression
	alias    string
}

func NewExpressionScan(fromExpr expression.Expression, alias string) *ExpressionScan {
	return &ExpressionScan{
		fromExpr: fromExpr,
		alias:    alias,
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

func (this *ExpressionScan) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "ExpressionScan"}
	r["expr"] = expression.NewStringer().Visit(this.fromExpr)
	r["alias"] = this.alias
	return json.Marshal(r)
}

func (this *ExpressionScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_        string `json:"#operator"`
		FromExpr string `json:"expr"`
		Alias    string `json:"alias"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	if _unmarshalled.FromExpr != "" {
		this.fromExpr, err = parser.Parse(_unmarshalled.FromExpr)
	}
	this.alias = _unmarshalled.Alias

	return err
}
