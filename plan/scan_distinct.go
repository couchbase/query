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

// DistinctScan scans multiple indexes and distincts the results.
type DistinctScan struct {
	readonly
	scan Operator
}

func NewDistinctScan(scan Operator) *DistinctScan {
	return &DistinctScan{
		scan: scan,
	}
}

func (this *DistinctScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDistinctScan(this)
}

func (this *DistinctScan) New() Operator {
	return &DistinctScan{}
}

func (this *DistinctScan) Scan() Operator {
	return this.scan
}

func (this *DistinctScan) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *DistinctScan) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "DistinctScan"}
	if f != nil {
		f(r)
	} else {
		r["scan"] = this.scan
	}
	return r
}

func (this *DistinctScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_    string          `json:"#operator"`
		Scan json.RawMessage `json:"scan"`
	}
	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	var scan_type struct {
		Operator string `json:"#operator"`
	}

	err = json.Unmarshal(_unmarshalled.Scan, &scan_type)
	if err != nil {
		return err
	}

	this.scan, err = MakeOperator(scan_type.Operator, _unmarshalled.Scan)
	return err
}
