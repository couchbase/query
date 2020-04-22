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

type Alias struct {
	readonly
	alias       string
	cost        float64
	cardinality float64
}

func NewAlias(alias string, cost, cardinality float64) *Alias {
	return &Alias{
		alias:       alias,
		cost:        cost,
		cardinality: cardinality,
	}
}

func (this *Alias) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAlias(this)
}

func (this *Alias) New() Operator {
	return &Alias{}
}

func (this *Alias) Alias() string {
	return this.alias
}

func (this *Alias) Cost() float64 {
	return this.cost
}

func (this *Alias) Cardinality() float64 {
	return this.cardinality
}

func (this *Alias) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Alias) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Alias"}
	r["as"] = this.alias
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

func (this *Alias) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string  `json:"#operator"`
		As          string  `json:"as"`
		Cost        float64 `json:"cost"`
		Cardinality float64 `json:"cardinality"`
	}
	err := json.Unmarshal(body, &_unmarshalled)
	this.alias = _unmarshalled.As
	this.cost = getCost(_unmarshalled.Cost)
	this.cardinality = getCardinality(_unmarshalled.Cardinality)
	return err
}
