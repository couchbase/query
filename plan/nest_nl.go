//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import (
	"encoding/json"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type NLNest struct {
	readonly
	optEstimate
	outer    bool
	alias    string
	onclause expression.Expression
	filter   expression.Expression
	child    Operator
}

func NewNLNest(nest *algebra.AnsiNest, child Operator, filter expression.Expression,
	cost, cardinality float64, size int64, frCost float64) *NLNest {
	rv := &NLNest{
		outer:    nest.Outer(),
		alias:    nest.Alias(),
		onclause: nest.Onclause(),
		child:    child,
		filter:   filter,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *NLNest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitNLNest(this)
}

func (this *NLNest) New() Operator {
	return &NLNest{}
}

func (this *NLNest) Outer() bool {
	return this.outer
}

func (this *NLNest) Alias() string {
	return this.alias
}

func (this *NLNest) Onclause() expression.Expression {
	return this.onclause
}

func (this *NLNest) SetOnclause(onclause expression.Expression) {
	this.onclause = onclause
}

func (this *NLNest) Child() Operator {
	return this.child
}

func (this *NLNest) Filter() expression.Expression {
	return this.filter
}

func (this *NLNest) SetFilter(filter expression.Expression) {
	this.filter = filter
}

func (this *NLNest) SetCardinality(cardinality float64) {
	this.cardinality = cardinality
}

func (this *NLNest) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *NLNest) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "NestedLoopNest"}
	r["alias"] = this.alias
	r["on_clause"] = this.onclause.String()

	if this.outer {
		r["outer"] = this.outer
	}

	if this.filter != nil {
		r["filter"] = this.filter.String()
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

func (this *NLNest) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string                 `json:"#operator"`
		Onclause    string                 `json:"on_clause"`
		Outer       bool                   `json:"outer"`
		Alias       string                 `json:"alias"`
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

	planContext := this.PlanContext()

	this.child, err = MakeOperator(child_type.Op_name, raw_child, planContext)
	if err != nil {
		return err
	}

	if planContext != nil {
		if this.onclause != nil {
			_, err = planContext.Map(this.onclause)
			if err != nil {
				return err
			}
		}
		if this.filter != nil {
			_, err = planContext.Map(this.filter)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (this *NLNest) verify(prepared *Prepared) errors.Error {
	return this.child.verify(prepared)
}
