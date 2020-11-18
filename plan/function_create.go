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
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/functions/resolver"
)

// Create function
type CreateFunction struct {
	ddl
	name    functions.FunctionName
	body    functions.FunctionBody
	replace bool
}

func NewCreateFunction(node *algebra.CreateFunction) *CreateFunction {
	return &CreateFunction{
		name:    node.Name(),
		body:    node.Body(),
		replace: node.Replace(),
	}
}

func (this *CreateFunction) Name() functions.FunctionName {
	return this.name
}

func (this *CreateFunction) Body() functions.FunctionBody {
	return this.body
}

func (this *CreateFunction) Replace() bool {
	return this.replace
}

func (this *CreateFunction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateFunction(this)
}

func (this *CreateFunction) New() Operator {
	return &CreateFunction{}
}

func (this *CreateFunction) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *CreateFunction) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "CreateFunction"}
	identity := make(map[string]interface{})
	definition := make(map[string]interface{})
	this.name.Signature(identity)
	this.body.Body(definition)
	r["identity"] = identity
	r["definition"] = definition
	r["replace"] = this.replace

	if f != nil {
		f(r)
	}
	return r
}

func (this *CreateFunction) UnmarshalJSON(bytes []byte) error {
	var _unmarshalled struct {
		_          string          `json:"#operator"`
		Identity   json.RawMessage `json:"identity"`
		Definition json.RawMessage `json:"definition"`
		Replace    bool            `json:"replace"`
	}
	var newErr errors.Error

	err := json.Unmarshal(bytes, &_unmarshalled)
	if err != nil {
		return err
	}

	this.name, err = makeName(_unmarshalled.Identity)
	if err != nil {
		return err
	}
	this.body, newErr = resolver.MakeBody(this.name.Name(), _unmarshalled.Definition)
	if newErr != nil {
		return newErr.GetICause()
	}
	this.replace = _unmarshalled.Replace
	return nil
}
