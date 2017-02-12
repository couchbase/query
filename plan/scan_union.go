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

	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

// UnionScan scans multiple indexes and unions the results.
type UnionScan struct {
	readonly
	scans []SecondaryScan
	limit expression.Expression
}

func NewUnionScan(limit expression.Expression, scans ...SecondaryScan) *UnionScan {
	return &UnionScan{
		scans: scans,
		limit: limit,
	}
}

func (this *UnionScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnionScan(this)
}

func (this *UnionScan) New() Operator {
	return &UnionScan{}
}

func (this *UnionScan) Covers() expression.Covers {
	if this.Covering() {
		return this.scans[0].Covers()
	} else {
		return nil
	}
}

func (this *UnionScan) FilterCovers() map[*expression.Cover]value.Value {
	if this.Covering() {
		return this.scans[0].FilterCovers()
	} else {
		return nil
	}
}

func (this *UnionScan) Covering() bool {
	for _, scan := range this.scans {
		if !scan.Covering() {
			return false
		}
	}

	return true
}

func (this *UnionScan) Scans() []SecondaryScan {
	return this.scans
}

func (this *UnionScan) Limit() expression.Expression {
	return this.limit
}

func (this *UnionScan) SetLimit(limit expression.Expression) {
	this.limit = limit

	for _, scan := range this.scans {
		if scan.Limit() != nil {
			scan.SetLimit(limit)
		}
	}
}

func (this *UnionScan) Streamline() SecondaryScan {
	scans := make([]SecondaryScan, 0, len(this.scans))
	hash := _STRING_SCANS_POOL.Get()
	defer _STRING_SCANS_POOL.Put(hash)

	for _, scan := range this.scans {
		s := scan.String()
		if _, ok := hash[s]; !ok {
			hash[s] = true
			scans = append(scans, scan)
		}
	}

	switch len(scans) {
	case 1:
		return scans[0]
	case len(this.scans):
		return this
	default:
		return NewUnionScan(this.limit, scans...)
	}
}

func (this *UnionScan) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}

func (this *UnionScan) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *UnionScan) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "UnionScan"}
	r["scans"] = this.scans

	if this.limit != nil {
		r["limit"] = expression.NewStringer().Visit(this.limit)
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *UnionScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string            `json:"#operator"`
		Scans []json.RawMessage `json:"scans"`
		Limit string            `json:"limit"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.scans = make([]SecondaryScan, 0, len(_unmarshalled.Scans))

	for _, raw_scan := range _unmarshalled.Scans {
		var scan_type struct {
			Operator string `json:"#operator"`
		}

		err = json.Unmarshal(raw_scan, &scan_type)
		if err != nil {
			return err
		}

		scan_op, err := MakeOperator(scan_type.Operator, raw_scan)
		if err != nil {
			return err
		}

		this.scans = append(this.scans, scan_op.(SecondaryScan))
	}

	if _unmarshalled.Limit != "" {
		this.limit, err = parser.Parse(_unmarshalled.Limit)
		if err != nil {
			return err
		}
	}

	return nil
}

var _STRING_SCANS_POOL = util.NewStringBoolPool(16)
