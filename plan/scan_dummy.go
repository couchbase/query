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

// DummyScan is used for SELECTs with no FROM clause.
type DummyScan struct {
	readonly

	cost        float64
	cardinality float64
}

func NewDummyScan(cost, cardinality float64) *DummyScan {
	return &DummyScan{
		cost:        cost,
		cardinality: cardinality,
	}
}

func (this *DummyScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDummyScan(this)
}

func (this *DummyScan) New() Operator {
	return &DummyScan{}
}

func (this *DummyScan) Cost() float64 {
	return this.cost
}

func (this *DummyScan) Cardinality() float64 {
	return this.cardinality
}

func (this *DummyScan) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *DummyScan) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "DummyScan"}
	if this.cost > 0.0 {
		r["cost"] = this.cost
	}
	if this.cardinality > 0.0 {
		r["cardinality"] = this.cardinality
	}
	if f != nil {
		f(r)
	}
	return r
}

func (this *DummyScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string  `json:"#operator"`
		Cost        float64 `json:"cost"`
		Cardinality float64 `json:"cardinality"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.cost = getCost(_unmarshalled.Cost)
	this.cardinality = getCardinality(_unmarshalled.Cardinality)

	return nil
}
