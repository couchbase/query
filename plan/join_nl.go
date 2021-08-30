//  Copyright 2017-Present Couchbase, Inc.
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

type NLJoin struct {
	readonly
	optEstimate
	outer     bool
	alias     string
	onclause  expression.Expression
	hintError string
	child     Operator
	filter    expression.Expression
}

func NewNLJoin(join *algebra.AnsiJoin, child Operator, filter expression.Expression,
	cost, cardinality float64, size int64, frCost float64) *NLJoin {
	rv := &NLJoin{
		outer:     join.Outer(),
		alias:     join.Alias(),
		onclause:  join.Onclause(),
		hintError: join.HintError(),
		child:     child,
		filter:    filter,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
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

func (this *NLJoin) SetOnclause(onclause expression.Expression) {
	this.onclause = onclause
}

func (this *NLJoin) HintError() string {
	return this.hintError
}

func (this *NLJoin) Child() Operator {
	return this.child
}

func (this *NLJoin) Filter() expression.Expression {
	return this.filter
}

func (this *NLJoin) SetFilter(filter expression.Expression) {
	this.filter = filter
}

func (this *NLJoin) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *NLJoin) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "NestedLoopJoin"}
	r["alias"] = this.alias
	if this.onclause != nil {
		r["on_clause"] = expression.NewStringer().Visit(this.onclause)
	}

	if this.outer {
		r["outer"] = this.outer
	}

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

func (this *NLJoin) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string                 `json:"#operator"`
		Onclause    string                 `json:"on_clause"`
		Outer       bool                   `json:"outer"`
		Alias       string                 `json:"alias"`
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
	this.alias = _unmarshalled.Alias
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

func (this *NLJoin) verify(prepared *Prepared) bool {
	return this.child.verify(prepared)
}
