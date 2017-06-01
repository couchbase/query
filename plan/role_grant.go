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

	"github.com/couchbase/query/algebra"
)

// Grant role
type GrantRole struct {
	readwrite
	node *algebra.GrantRole
}

func NewGrantRole(node *algebra.GrantRole) *GrantRole {
	return &GrantRole{
		node: node,
	}
}

func (this *GrantRole) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitGrantRole(this)
}

func (this *GrantRole) New() Operator {
	return &GrantRole{}
}

func (this *GrantRole) Node() *algebra.GrantRole {
	return this.node
}

func (this *GrantRole) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *GrantRole) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "GrantRole"}
	r["roles"] = this.node.Roles()
	r["keyspaces"] = this.node.Keyspaces()
	r["users"] = this.node.Users()
	if f != nil {
		f(r)
	}
	return r
}

func (this *GrantRole) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string   `json:"#operator"`
		Roles     []string `json:"roles"`
		Keyspaces []string `json:"keyspaces"`
		Users     []string `json:"users"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.node = algebra.NewGrantRole(_unmarshalled.Roles, _unmarshalled.Keyspaces, _unmarshalled.Users)
	return nil
}
