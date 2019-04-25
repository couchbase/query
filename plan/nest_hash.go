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

type HashNest struct {
	readonly
	outer       bool
	onclause    expression.Expression
	child       Operator
	buildExprs  expression.Expressions
	probeExprs  expression.Expressions
	buildAlias  string
	hintError   string
	cost        float64
	cardinality float64
}

func NewHashNest(nest *algebra.AnsiNest, child Operator, buildExprs, probeExprs expression.Expressions,
	buildAlias string, cost, cardinality float64) *HashNest {
	return &HashNest{
		outer:       nest.Outer(),
		onclause:    nest.Onclause(),
		child:       child,
		buildExprs:  buildExprs,
		probeExprs:  probeExprs,
		buildAlias:  buildAlias,
		hintError:   nest.HintError(),
		cost:        cost,
		cardinality: cardinality,
	}
}

func (this *HashNest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitHashNest(this)
}

func (this *HashNest) New() Operator {
	return &HashNest{}
}

func (this *HashNest) Outer() bool {
	return this.outer
}

func (this *HashNest) Onclause() expression.Expression {
	return this.onclause
}

func (this *HashNest) Child() Operator {
	return this.child
}

func (this *HashNest) BuildExprs() expression.Expressions {
	return this.buildExprs
}

func (this *HashNest) ProbeExprs() expression.Expressions {
	return this.probeExprs
}

func (this *HashNest) BuildAlias() string {
	return this.buildAlias
}

func (this *HashNest) HintError() string {
	return this.hintError
}

func (this *HashNest) Cost() float64 {
	return this.cost
}

func (this *HashNest) Cardinality() float64 {
	return this.cardinality
}

func (this *HashNest) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *HashNest) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "HashNest"}
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

	r["build_alias"] = this.buildAlias

	if this.hintError != "" {
		r["hint_not_followed"] = this.hintError
	}

	if this.cost > 0.0 {
		r["cost"] = this.cost
	}

	if this.cardinality > 0.0 {
		r["cardinality"] = this.cardinality
	}

	r["~child"] = this.child

	if f != nil {
		f(r)
	}
	return r
}

func (this *HashNest) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string          `json:"#operator"`
		Onclause    string          `json:"on_clause"`
		Outer       bool            `json:"outer"`
		BuildExprs  []string        `json:"build_exprs"`
		ProbeExprs  []string        `json:"probe_exprs"`
		BuildAlias  string          `json:"build_alias"`
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

	this.buildAlias = _unmarshalled.BuildAlias
	this.hintError = _unmarshalled.HintError

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

func (this *HashNest) verify(prepared *Prepared) bool {
	return this.child.verify(prepared)
}
