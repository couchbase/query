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

type IndexProjection struct {
	EntryKeys  []int
	PrimaryKey bool
}

func NewIndexProjection(size int, primary bool) *IndexProjection {
	return &IndexProjection{
		EntryKeys:  make([]int, 0, size),
		PrimaryKey: primary,
	}
}

func (this *IndexProjection) Copy() *IndexProjection {
	return &IndexProjection{
		EntryKeys:  this.EntryKeys,
		PrimaryKey: this.PrimaryKey,
	}
}

func (this *IndexProjection) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *IndexProjection) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := make(map[string]interface{}, 2)

	if len(this.EntryKeys) != 0 {
		r["entry_keys"] = this.EntryKeys
	}
	if this.PrimaryKey {
		r["primary_key"] = this.PrimaryKey
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *IndexProjection) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		EntryKeys  []int `json:"entry_keys"`
		PrimaryKey bool  `json:"primary_key"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.EntryKeys = _unmarshalled.EntryKeys
	this.PrimaryKey = _unmarshalled.PrimaryKey

	return nil
}

func (this *IndexProjection) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}
