//  Copyright (c) 2018 Couchbase, Inc.
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

type With struct {
	readonly
	optEstimate
	bindings expression.Bindings
	child    Operator
}

func NewWith(bindings expression.Bindings, child Operator, cost, cardinality float64) *With {
	rv := &With{
		bindings: bindings,
		child:    child,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality)
	return rv
}

func (this *With) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitWith(this)
}

func (this *With) New() Operator {
	return &With{}
}

func (this *With) Bindings() expression.Bindings {
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
		_           string             `json:"#operator"`
		Bindings    json.RawMessage    `json:"bindings"`
		Child       json.RawMessage    `json:"~child"`
		OptEstimate map[string]float64 `json:"optimizer_estimates"`
	}

	var child_type struct {
		Operator string `json:"#operator"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.bindings, err = unmarshal.UnmarshalBindings(_unmarshalled.Bindings)

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
