//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import (
	"encoding/json"

	"github.com/couchbase/query/errors"
)

type UnionAll struct {
	readonly
	optEstimate
	children []Operator
}

func NewUnionAll(cost, cardinality float64, size int64, frCost float64, children ...Operator) *UnionAll {
	rv := &UnionAll{
		children: children,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *UnionAll) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnionAll(this)
}

func (this *UnionAll) New() Operator {
	return &UnionAll{}
}

func (this *UnionAll) Children() []Operator {
	return this.children
}

func (this *UnionAll) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *UnionAll) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "UnionAll"}
	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}
	if f != nil {
		f(r)
	} else {
		r["~children"] = this.children
	}
	return r
}

func (this *UnionAll) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string                 `json:"#operator"`
		Children    []json.RawMessage      `json:"~children"`
		OptEstimate map[string]interface{} `json:"optimizer_estimates"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	planContext := this.PlanContext()

	this.children = make([]Operator, 0, len(_unmarshalled.Children))

	for _, raw_child := range _unmarshalled.Children {
		var child_type struct {
			Op_name string `json:"#operator"`
		}

		err = json.Unmarshal(raw_child, &child_type)
		if err != nil {
			return err
		}

		child_op, err := MakeOperator(child_type.Op_name, raw_child, planContext)
		if err != nil {
			return err
		}

		this.children = append(this.children, child_op)
	}

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

	return err
}

func (this *UnionAll) verify(prepared *Prepared) errors.Error {
	for _, child := range this.children {
		if err := child.verify(prepared); err != nil {
			return err
		}
	}

	return nil
}
