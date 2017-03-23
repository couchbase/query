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

	"github.com/couchbase/query/auth"
)

type Authorize struct {
	readonly
	privs *auth.Privileges `json:"privileges"`
	child Operator         `json:"~child"`
}

func NewAuthorize(privs *auth.Privileges, child Operator) *Authorize {
	return &Authorize{
		privs: privs,
		child: child,
	}
}

func (this *Authorize) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAuthorize(this)
}

func (this *Authorize) New() Operator {
	return &Authorize{}
}

func (this *Authorize) Privileges() *auth.Privileges {
	return this.privs
}

func (this *Authorize) Readonly() bool {
	return this.child.Readonly()
}

func (this *Authorize) Child() Operator {
	return this.child
}

func (this *Authorize) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Authorize) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Authorize"}
	r["privileges"] = this.privs
	if f != nil {
		f(r)
	} else {
		r["~child"] = this.child
	}
	return r
}

func (this *Authorize) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string           `json:"#operator"`
		Privs *auth.Privileges `json:"privileges"`
		Child json.RawMessage  `json:"~child"`
	}
	var child_type struct {
		Operator string `json:"#operator"`
	}
	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}
	this.privs = _unmarshalled.Privs

	err = json.Unmarshal(_unmarshalled.Child, &child_type)
	if err != nil {
		return err
	}
	this.child, err = MakeOperator(child_type.Operator, _unmarshalled.Child)
	return err
}
