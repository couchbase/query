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
)

type UnionAll struct {
	readonly
	children    []Operator
	cost        float64
	cardinality float64
}

func NewUnionAll(cost, cardinality float64, children ...Operator) *UnionAll {
	return &UnionAll{
		children:    children,
		cost:        cost,
		cardinality: cardinality,
	}
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

func (this *UnionAll) Cost() float64 {
	return this.cost
}

func (this *UnionAll) Cardinality() float64 {
	return this.cardinality
}

func (this *UnionAll) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *UnionAll) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "UnionAll"}
	if this.cost > 0.0 {
		r["cost"] = this.cost
	}
	if this.cardinality > 0.0 {
		r["cardinality"] = this.cardinality
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
		_           string            `json:"#operator"`
		Children    []json.RawMessage `json:"~children"`
		Cost        float64           `json:"cost"`
		Cardinality float64           `json:"cardinality"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.children = make([]Operator, 0, len(_unmarshalled.Children))

	for _, raw_child := range _unmarshalled.Children {
		var child_type struct {
			Op_name string `json:"#operator"`
		}

		err = json.Unmarshal(raw_child, &child_type)
		if err != nil {
			return err
		}

		child_op, err := MakeOperator(child_type.Op_name, raw_child)
		if err != nil {
			return err
		}

		this.children = append(this.children, child_op)
	}

	this.cost = getCost(_unmarshalled.Cost)
	this.cardinality = getCardinality(_unmarshalled.Cardinality)

	return err
}

func (this *UnionAll) verify(prepared *Prepared) bool {
	for _, child := range this.children {
		if !child.verify(prepared) {
			return false
		}
	}

	return true
}
