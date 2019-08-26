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
	scans       []SecondaryScan
	limit       expression.Expression
	offset      expression.Expression
	cost        float64
	cardinality float64
}

func NewUnionScan(limit, offset expression.Expression, cost, cardinality float64, scans ...SecondaryScan) *UnionScan {
	for _, scan := range scans {
		if scan.Limit() == nil {
			limit = nil
			offset = nil
			break
		}
	}

	return &UnionScan{
		scans:       scans,
		limit:       limit,
		offset:      offset,
		cost:        cost,
		cardinality: cardinality,
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

func (this *UnionScan) Offset() expression.Expression {
	return this.offset
}

func (this *UnionScan) SetLimit(limit expression.Expression) {
	this.limit = limit

	for _, scan := range this.scans {
		if scan.Limit() != nil {
			scan.SetLimit(limit)
		}
	}
}

func (this *UnionScan) SetOffset(offset expression.Expression) {
	this.offset = offset

	for _, scan := range this.scans {
		scan.SetOffset(offset)
	}
}

func (this *UnionScan) Cost() float64 {
	return this.cost
}

func (this *UnionScan) Cardinality() float64 {
	return this.cardinality
}

func (this *UnionScan) CoverJoinSpanExpressions(coverer *expression.Coverer) error {
	for _, scan := range this.scans {
		err := scan.CoverJoinSpanExpressions(coverer)
		if err != nil {
			return err
		}
	}

	return nil
}

func (this *UnionScan) GroupAggs() *IndexGroupAggregates {
	return nil
}

func (this *UnionScan) OrderTerms() IndexKeyOrders {
	return nil
}

func (this *UnionScan) SetCovers(covers expression.Covers) {
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
		scan := NewUnionScan(this.limit, this.offset, this.cost, this.cardinality, scans...)
		this.limit = scan.Limit()
		this.offset = scan.Offset()
		return scan
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

	if this.offset != nil {
		r["offset"] = expression.NewStringer().Visit(this.offset)
	}

	if this.cost > 0.0 {
		r["cost"] = this.cost
	}

	if this.cardinality > 0.0 {
		r["cardinality"] = this.cardinality
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *UnionScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string            `json:"#operator"`
		Scans       []json.RawMessage `json:"scans"`
		Limit       string            `json:"limit"`
		Offset      string            `json:"offset"`
		Cost        float64           `json:"cost"`
		Cardinality float64           `json:"cardinality"`
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

	if _unmarshalled.Offset != "" {
		this.offset, err = parser.Parse(_unmarshalled.Offset)
		if err != nil {
			return err
		}
	}

	if _unmarshalled.Cost > 0.0 {
		this.cost = _unmarshalled.Cost
	} else {
		this.cost = PLAN_COST_NOT_AVAIL
	}

	if _unmarshalled.Cardinality > 0.0 {
		this.cardinality = _unmarshalled.Cardinality
	} else {
		this.cardinality = PLAN_CARD_NOT_AVAIL
	}

	return nil
}

func (this *UnionScan) verify(prepared *Prepared) bool {
	for _, scan := range this.scans {
		if !scan.verify(prepared) {
			return false
		}
	}

	return true
}

var _STRING_SCANS_POOL = util.NewStringBoolPool(16)
