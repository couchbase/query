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

	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/unmarshal"
)

type Let struct {
	readonly
	bindings    expression.Bindings
	cost        float64
	cardinality float64
}

func NewLet(bindings expression.Bindings, cost, cardinality float64) *Let {
	return &Let{
		bindings:    bindings,
		cost:        cost,
		cardinality: cardinality,
	}
}

func (this *Let) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitLet(this)
}

func (this *Let) New() Operator {
	return &Let{}
}

func (this *Let) Bindings() expression.Bindings {
	return this.bindings
}

func (this *Let) Cost() float64 {
	return this.cost
}

func (this *Let) Cardinality() float64 {
	return this.cardinality
}

func (this *Let) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Let) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Let"}
	r["bindings"] = this.bindings
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

func (this *Let) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string          `json:"#operator"`
		Bindings    json.RawMessage `json:"bindings"`
		Cost        float64         `json:"cost"`
		Cardinality float64         `json:"cardinality"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.bindings, err = unmarshal.UnmarshalBindings(_unmarshalled.Bindings)
	if err != nil {
		return err
	}

	this.cost = getCost(_unmarshalled.Cost)
	this.cardinality = getCardinality(_unmarshalled.Cardinality)

	return nil
}
