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

// IntersectScan scans multiple indexes and intersects the results.
type IntersectScan struct {
	readonly
	scans []Operator
}

func NewIntersectScan(scans ...Operator) *IntersectScan {
	return &IntersectScan{
		scans: scans,
	}
}

func (this *IntersectScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIntersectScan(this)
}

func (this *IntersectScan) New() Operator {
	return &IntersectScan{}
}

func (this *IntersectScan) Scans() []Operator {
	return this.scans
}

func (this *IntersectScan) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "IntersectScan"}

	// FIXME
	r["scans"] = this.scans

	return json.Marshal(r)
}

func (this *IntersectScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string            `json:"#operator"`
		Scans []json.RawMessage `json:"scans"`
	}
	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.scans = []Operator{}

	for _, raw_scan := range _unmarshalled.Scans {
		var scan_type struct {
			Operator string `json:"#operator"`
		}
		var read_only struct {
			Readonly bool `json:"readonly"`
		}
		err = json.Unmarshal(raw_scan, &scan_type)
		if err != nil {
			return err
		}

		if scan_type.Operator == "" {
			err = json.Unmarshal(raw_scan, &read_only)
			if err != nil {
				return err
			} else {
				// This should be a readonly object
			}
		} else {
			scan_op, err := MakeOperator(scan_type.Operator, raw_scan)
			if err != nil {
				return err
			}

			this.scans = append(this.scans, scan_op)
		}
	}

	return err
}
