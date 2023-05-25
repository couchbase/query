//  Copyright 2018-Present Couchbase, Inc.
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
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type With struct {
	readonly
	optEstimate
	bindings expression.Withs
	child    Operator
}

func NewWith(bindings expression.Withs, child Operator, cost, cardinality float64,
	size int64, frCost float64) *With {
	rv := &With{
		bindings: bindings,
		child:    child,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *With) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitWith(this)
}

func (this *With) New() Operator {
	return &With{}
}

func (this *With) Bindings() expression.Withs {
	return this.bindings
}

func (this *With) Readonly() bool {
	return this.child.Readonly()
}

func (this *With) Child() Operator {
	return this.child
}

func (this *With) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *With) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "With"}
	r["bindings"] = this.bindings
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

func (this *With) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string                 `json:"#operator"`
		Bindings    json.RawMessage        `json:"bindings"`
		Child       json.RawMessage        `json:"~child"`
		OptEstimate map[string]interface{} `json:"optimizer_estimates"`
	}

	var child_type struct {
		Operator string `json:"#operator"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.bindings, err = unmarshalWiths(_unmarshalled.Bindings)

	err = json.Unmarshal(_unmarshalled.Child, &child_type)
	if err != nil {
		return err
	}
	this.child, err = MakeOperator(child_type.Operator, _unmarshalled.Child)
	if err != nil {
		return err
	}

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

	return nil
}

func unmarshalWiths(body []byte) (expression.Withs, error) {
	var _unmarshalled []struct {
		Alias string `json:"alias"`
		Expr  string `json:"expr"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return nil, err
	}

	withs := make(expression.Withs, len(_unmarshalled))
	for i, with := range _unmarshalled {
		expr, err := parser.Parse(with.Expr)
		if err != nil {
			return nil, err
		}

		withs[i] = algebra.NewWith(with.Alias, expr)
	}

	return withs, nil
}
