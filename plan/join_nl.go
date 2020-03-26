//  Copyright (c) 2017 Couchbase, Inc.
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
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type NLJoin struct {
	readonly
	outer       bool
	alias       string
	onclause    expression.Expression
	hintError   string
	child       Operator
	cost        float64
	cardinality float64
}

func NewNLJoin(join *algebra.AnsiJoin, child Operator, cost, cardinality float64) *NLJoin {
	rv := &NLJoin{
		outer:       join.Outer(),
		alias:       join.Alias(),
		onclause:    join.Onclause(),
		hintError:   join.HintError(),
		child:       child,
		cost:        cost,
		cardinality: cardinality,
	}

	return rv
}

func (this *NLJoin) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitNLJoin(this)
}

func (this *NLJoin) New() Operator {
	return &NLJoin{}
}

func (this *NLJoin) Outer() bool {
	return this.outer
}

func (this *NLJoin) Alias() string {
	return this.alias
}

func (this *NLJoin) Onclause() expression.Expression {
	return this.onclause
}

func (this *NLJoin) HintError() string {
	return this.hintError
}

func (this *NLJoin) Child() Operator {
	return this.child
}

func (this *NLJoin) Cost() float64 {
	return this.cost
}

func (this *NLJoin) Cardinality() float64 {
	return this.cardinality
}

func (this *NLJoin) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *NLJoin) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "NestedLoopJoin"}
	r["alias"] = this.alias
	r["on_clause"] = expression.NewStringer().Visit(this.onclause)

	if this.outer {
		r["outer"] = this.outer
	}

	if this.hintError != "" {
		r["hint_not_followed"] = this.hintError
	}

	if this.cost > 0.0 {
		r["cost"] = this.cost
	}

	if this.cardinality > 0.0 {
		r["cardinality"] = this.cardinality
	}

	if f != nil {
		f(r)
	} else {
		r["~child"] = this.child
	}
	return r
}

func (this *NLJoin) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string          `json:"#operator"`
		Onclause    string          `json:"on_clause"`
		Outer       bool            `json:"outer"`
		Alias       string          `json:"alias"`
		HintError   string          `json:"hint_not_followed"`
		Cost        float64         `json:"cost"`
		Cardinality float64         `json:"cardinality"`
		Child       json.RawMessage `json:"~child"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	if _unmarshalled.Onclause != "" {
		this.onclause, err = parser.Parse(_unmarshalled.Onclause)
		if err != nil {
			return err
		}
	}

	this.outer = _unmarshalled.Outer
	this.alias = _unmarshalled.Alias
	this.hintError = _unmarshalled.HintError

	this.cost = getCost(_unmarshalled.Cost)
	this.cardinality = getCardinality(_unmarshalled.Cardinality)

	raw_child := _unmarshalled.Child
	var child_type struct {
		Op_name string `json:"#operator"`
	}

	err = json.Unmarshal(raw_child, &child_type)
	if err != nil {
		return err
	}

	this.child, err = MakeOperator(child_type.Op_name, raw_child)
	if err != nil {
		return err
	}

	return nil
}

func (this *NLJoin) verify(prepared *Prepared) bool {
	return this.child.verify(prepared)
}
