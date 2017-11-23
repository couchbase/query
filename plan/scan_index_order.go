//  Copyright (c) 2017 Couchbase, Inc.
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

type IndexKeyOrders []*IndexKeyOrder

type IndexKeyOrder struct {
	KeyPos int
	Desc   bool
}

func NewIndexKeyOrders(keyPos int, desc bool) *IndexKeyOrder {
	return &IndexKeyOrder{
		KeyPos: keyPos,
		Desc:   desc,
	}
}

func (this *IndexKeyOrder) Copy() *IndexKeyOrder {
	return &IndexKeyOrder{
		KeyPos: this.KeyPos,
		Desc:   this.Desc,
	}
}

func (this *IndexKeyOrder) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *IndexKeyOrder) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := make(map[string]interface{}, 2)

	r["keypos"] = this.KeyPos
	if this.Desc {
		r["desc"] = this.Desc
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *IndexKeyOrder) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		KeyPos int  `json:"keypos"`
		Desc   bool `json:"desc"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.KeyPos = _unmarshalled.KeyPos
	this.Desc = _unmarshalled.Desc

	return nil
}

func (this *IndexKeyOrder) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}

func (this IndexKeyOrders) Copy() IndexKeyOrders {
	orderkeys := make(IndexKeyOrders, len(this))
	for i, r := range this {
		if r != nil {
			orderkeys[i] = r.Copy()
		}
	}

	return orderkeys
}
