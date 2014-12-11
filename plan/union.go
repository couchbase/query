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

type UnionAll struct {
	readonly
	children []Operator
}

func NewUnionAll(children ...Operator) *UnionAll {
	return &UnionAll{
		children: children,
	}
}

func (this *UnionAll) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnionAll(this)
}

func (this *UnionAll) New() Operator {
	return &UnionAll{}
}

func (this *UnionAll) Children() []Operator {
	return this.children
}

func (this *UnionAll) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "UnionAll"}
	r["children"] = this.children
	return json.Marshal(r)
}

func (this *UnionAll) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_        string            `json:"#operator"`
		Children []json.RawMessage `json:"children"`
	}
	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.children = []Operator{}

	for _, raw_child := range _unmarshalled.Children {
		var child_type struct {
			Op_name string `json:"#operator"`
		}
		var read_only struct {
			Readonly bool `json:"readonly"`
		}
		err = json.Unmarshal(raw_child, &child_type)
		if err != nil {
			return err
		}

		if child_type.Op_name == "" {
			err = json.Unmarshal(raw_child, &read_only)
			if err != nil {
				return err
			} else {
				// This should be a readonly object
			}
		} else {
			child_op, err := MakeOperator(child_type.Op_name, raw_child)
			if err != nil {
				return err
			}
			this.children = append(this.children, child_op)
		}
	}

	return err
}
