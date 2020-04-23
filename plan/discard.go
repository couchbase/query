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

type Discard struct {
	readonly

	cost        float64
	cardinality float64
}

func NewDiscard(cost, cardinality float64) *Discard {
	return &Discard{
		cost:        cost,
		cardinality: cardinality,
	}
}

func (this *Discard) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDiscard(this)
}

func (this *Discard) New() Operator {
	return &Discard{}
}

func (this *Discard) Cost() float64 {
	return this.cost
}

func (this *Discard) Cardinality() float64 {
	return this.cardinality
}

func (this *Discard) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Discard) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Discard"}
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

func (this *Discard) UnmarshalJSON(body []byte) error {
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
