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

type Explain struct {
	execution
	op   Operator
	text string
}

func NewExplain(op Operator, text string) *Explain {
	return &Explain{
		op:   op,
		text: text,
	}
}

func (this *Explain) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExplain(this)
}

func (this *Explain) New() Operator {
	return &Explain{}
}

func (this *Explain) Operator() Operator {
	return this.op
}

func (this *Explain) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Explain) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := make(map[string]interface{}, 2)
	r["text"] = this.text
	if this.op != nil {
		if this.op.Cost() > 0.0 {
			r["cost"] = this.op.Cost()
		}
		if this.op.Cardinality() > 0.0 {
			r["cardinality"] = this.op.Cardinality()
		}
	}
	if f != nil {
		f(r)
	} else {
		r["plan"] = this.op
	}
	return r
}

func (this *Explain) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		Op          json.RawMessage `json:"plan"`
		Text        string          `json:"text"`
		Cost        float64         `json:"cost"`
		Cardinality float64         `json:"cardinality"`
	}

	var op_type struct {
		Operator string `json:"#operator"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	err = json.Unmarshal(_unmarshalled.Op, &op_type)
	if err != nil {
		return err
	}

	this.text = _unmarshalled.Text

	err = json.Unmarshal(_unmarshalled.Op, &op_type)
	if err != nil {
		return err
	}

	// Cost/cardinality is included in the explain plan so it's easy to
	// see the overall cost/cardinality for the entire plan, there is
	// no need to put the info anywhere

	this.op, err = MakeOperator(op_type.Operator, _unmarshalled.Op)
	return err
}
