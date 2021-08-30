//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package plan

import (
	"encoding/json"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type HashNest struct {
	readonly
	optEstimate
	outer      bool
	onclause   expression.Expression
	child      Operator
	buildExprs expression.Expressions
	probeExprs expression.Expressions
	buildAlias string
	hintError  string
	filter     expression.Expression
}

func NewHashNest(nest *algebra.AnsiNest, child Operator, buildExprs, probeExprs expression.Expressions,
	buildAlias string, filter expression.Expression, cost, cardinality float64,
	size int64, frCost float64) *HashNest {
	rv := &HashNest{
		outer:      nest.Outer(),
		onclause:   nest.Onclause(),
		child:      child,
		buildExprs: buildExprs,
		probeExprs: probeExprs,
		buildAlias: buildAlias,
		hintError:  nest.HintError(),
		filter:     filter,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
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

func (this *HashNest) SetOnclause(onclause expression.Expression) {
	this.onclause = onclause
}

func (this *HashNest) Child() Operator {
	return this.child
}

func (this *HashNest) BuildExprs() expression.Expressions {
	return this.buildExprs
}

func (this *HashNest) SetBuildExprs(buildExprs expression.Expressions) {
	this.buildExprs = buildExprs
}

func (this *HashNest) ProbeExprs() expression.Expressions {
	return this.probeExprs
}

func (this *HashNest) SetProbeExprs(probeExprs expression.Expressions) {
	this.probeExprs = probeExprs
}

func (this *HashNest) BuildAlias() string {
	return this.buildAlias
}

func (this *HashNest) HintError() string {
	return this.hintError
}

func (this *HashNest) Filter() expression.Expression {
	return this.filter
}

func (this *HashNest) SetFilter(filter expression.Expression) {
	this.filter = filter
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

	if this.filter != nil {
		r["filter"] = expression.NewStringer().Visit(this.filter)
	}

	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}

	if f != nil {
		f(r)
	} else {
		r["~child"] = this.child
	}
	return r
}

func (this *HashNest) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string                 `json:"#operator"`
		Onclause    string                 `json:"on_clause"`
		Outer       bool                   `json:"outer"`
		BuildExprs  []string               `json:"build_exprs"`
		ProbeExprs  []string               `json:"probe_exprs"`
		BuildAlias  string                 `json:"build_alias"`
		HintError   string                 `json:"hint_not_followed"`
		Filter      string                 `json:"filter"`
		OptEstimate map[string]interface{} `json:"optimizer_estimates"`
		Child       json.RawMessage        `json:"~child"`
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

	if _unmarshalled.Filter != "" {
		this.filter, err = parser.Parse(_unmarshalled.Filter)
		if err != nil {
			return err
		}
	}

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

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
