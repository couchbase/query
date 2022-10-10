//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

//go:build !enterprise

package plan

import (
	"encoding/json"
)

type IndexAdvice struct {
	readonly
}

func NewIndexAdvice() *IndexAdvice {
	return &IndexAdvice{}
}

func (this *IndexAdvice) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexAdvice(this)
}

func (this *IndexAdvice) New() Operator {
	return &IndexAdvice{}
}

func (this *IndexAdvice) Operator() Operator {
	return this
}

func (this *IndexAdvice) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *IndexAdvice) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "IndexAdvice"}

	if f != nil {
		f(r)
	}
	return r
}

func (this *IndexAdvice) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_ string `json:"#operator"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	return nil
}
