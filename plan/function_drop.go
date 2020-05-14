//  Copyright (c) 2019 Couchbase, Inc.
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

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/functions"
)

// Drop function
type DropFunction struct {
	ddl
	name functions.FunctionName
}

func NewDropFunction(node *algebra.DropFunction) *DropFunction {
	return &DropFunction{
		name: node.Name(),
	}
}

func (this *DropFunction) Name() functions.FunctionName {
	return this.name
}

func (this *DropFunction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropFunction(this)
}

func (this *DropFunction) New() Operator {
	return &DropFunction{}
}

func (this *DropFunction) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *DropFunction) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "DropFunction"}
	identity := make(map[string]interface{})
	this.name.Signature(identity)
	r["identity"] = identity

	if f != nil {
		f(r)
	}
	return r
}

func (this *DropFunction) UnmarshalJSON(bytes []byte) error {
	var _unmarshalled struct {
		_        string          `json:"#operator"`
		Identity json.RawMessage `json:"identity"`
	}

	err := json.Unmarshal(bytes, &_unmarshalled)
	if err != nil {
		return err
	}

	this.name, err = makeName(_unmarshalled.Identity)
	if err != nil {
		return err
	}
	return nil
}
