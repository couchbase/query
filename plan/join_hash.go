//  Copyright (c) 2018 Couchbase, Inc.
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

type HashJoin struct {
	readonly
	outer        bool
	onclause     expression.Expression
	child        Operator
	buildExprs   expression.Expressions
	probeExprs   expression.Expressions
	buildAliases []string
	hintError    string
	filter       expression.Expression
	cost         float64
	cardinality  float64
}

func NewHashJoin(join *algebra.AnsiJoin, child Operator, buildExprs, probeExprs expression.Expressions,
	buildAliases []string, filter expression.Expression, cost, cardinality float64) *HashJoin {
	return &HashJoin{
		outer:        join.Outer(),
		onclause:     join.Onclause(),
		child:        child,
		buildExprs:   buildExprs,
		probeExprs:   probeExprs,
		buildAliases: buildAliases,
		hintError:    join.HintError(),
		filter:       filter,
		cost:         cost,
		cardinality:  cardinality,
	}
}

func (this *HashJoin) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitHashJoin(this)
}

func (this *HashJoin) New() Operator {
	return &HashJoin{}
}

func (this *HashJoin) Outer() bool {
	return this.outer
}

func (this *HashJoin) Onclause() expression.Expression {
	return this.onclause
}

func (this *HashJoin) Child() Operator {
	return this.child
}

func (this *HashJoin) BuildExprs() expression.Expressions {
	return this.buildExprs
}

func (this *HashJoin) ProbeExprs() expression.Expressions {
	return this.probeExprs
}

func (this *HashJoin) BuildAliases() []string {
	return this.buildAliases
}

func (this *HashJoin) HintError() string {
	return this.hintError
}

func (this *HashJoin) Filter() expression.Expression {
	return this.filter
}

func (this *HashJoin) Cost() float64 {
	return this.cost
}

func (this *HashJoin) Cardinality() float64 {
	return this.cardinality
}

func (this *HashJoin) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *HashJoin) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "HashJoin"}
	r["on_clause"] = expression.NewStringer().Visit(this.onclause)

	if this.outer {
		r["outer"] = this.outer
	}

	buildList := make([]string, 0, len(this.buildExprs))
	for _, build := range this.buildExprs {
		buildList = append(buildList, expression.NewStringer().Visit(build))
	}
	r["build_exprs"] = buildList

	probeList := make([]string, 0, len(this.probeExprs))
	for _, probe := range this.probeExprs {
		probeList = append(probeList, expression.NewStringer().Visit(probe))
	}
	r["probe_exprs"] = probeList

	r["build_aliases"] = this.buildAliases

	if this.hintError != "" {
		r["hint_not_followed"] = this.hintError
	}

	if this.filter != nil {
		r["filter"] = expression.NewStringer().Visit(this.filter)
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

func (this *HashJoin) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_            string          `json:"#operator"`
		Onclause     string          `json:"on_clause"`
		Outer        bool            `json:"outer"`
		BuildExprs   []string        `json:"build_exprs"`
		ProbeExprs   []string        `json:"probe_exprs"`
		BuildAliases []string        `json:"build_aliases"`
		HintError    string          `json:"hint_not_followed"`
		Filter       string          `json:"filter"`
		Cost         float64         `json:"cost"`
		Cardinality  float64         `json:"cardinality"`
		Child        json.RawMessage `json:"~child"`
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

	this.buildExprs = make(expression.Expressions, len(_unmarshalled.BuildExprs))
	for i, build := range _unmarshalled.BuildExprs {
		buildExpr, err := parser.Parse(build)
		if err != nil {
			return err
		}
		this.buildExprs[i] = buildExpr
	}

	this.probeExprs = make(expression.Expressions, len(_unmarshalled.ProbeExprs))
	for i, probe := range _unmarshalled.ProbeExprs {
		probeExpr, err := parser.Parse(probe)
		if err != nil {
			return err
		}
		this.probeExprs[i] = probeExpr
	}

	this.buildAliases = _unmarshalled.BuildAliases
	this.hintError = _unmarshalled.HintError

	if _unmarshalled.Filter != "" {
		this.filter, err = parser.Parse(_unmarshalled.Filter)
		if err != nil {
			return err
		}
	}

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

func (this *HashJoin) verify(prepared *Prepared) bool {
	return this.child.verify(prepared)
}
