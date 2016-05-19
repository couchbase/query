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

	"github.com/couchbase/query/value"
)

type Prepare struct {
	readonly
	prepared value.Value
}

func NewPrepare(prepared value.Value) *Prepare {
	return &Prepare{
		prepared: prepared,
	}
}

func (this *Prepare) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitPrepare(this)
}

func (this *Prepare) New() Operator {
	return &Prepare{}
}

func (this *Prepare) Prepared() value.Value {
	return this.prepared
}

func (this *Prepare) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "Prepare"}
	r["prepared"] = this.prepared
	return json.Marshal(r)
}

func (this *Prepare) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_        string          `json:"#operator"`
		Prepared json.RawMessage `json:"prepared"`
	}
	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.prepared = value.NewValue(_unmarshalled.Prepared)
	return nil
}
