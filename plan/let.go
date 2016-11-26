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
	bindings expression.Bindings
}

func NewLet(bindings expression.Bindings) *Let {
	return &Let{
		bindings: bindings,
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

func (this *Let) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Let) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Let"}
	r["bindings"] = this.bindings
	if f != nil {
		f(r)
	}
	return r
}

func (this *Let) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_        string          `json:"#operator"`
		Bindings json.RawMessage `json:"bindings"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.bindings, err = unmarshal.UnmarshalBindings(_unmarshalled.Bindings)
	return err
}
