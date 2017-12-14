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
	"github.com/couchbase/query/value"
)

// IntersectScan that preserves index order of first scan.
type OrderedIntersectScan struct {
	readonly
	scans []SecondaryScan
	limit expression.Expression
}

func NewOrderedIntersectScan(limit expression.Expression, scans ...SecondaryScan) *OrderedIntersectScan {
	for _, scan := range scans {
		if scan.Limit() != nil {
			scan.SetLimit(nil)
		}
		scan.SetOffset(nil)
	}

	buf := make([]SecondaryScan, 0, 2*len(scans))
	buf = flattenOrderedIntersectScans(scans[0], buf)
	scans = append(buf, flattenIntersectScans(scans[1:], buf[len(buf):])...)

	n := len(scans)
	if n > 64 {
		return NewOrderedIntersectScan(
			limit,
			scans[0],
			NewIntersectScan(nil, scans[1:n/2]...),
			NewIntersectScan(nil, scans[n/2:]...),
		)
	}

	return &OrderedIntersectScan{
		scans: scans,
		limit: limit,
	}
}

func (this *OrderedIntersectScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitOrderedIntersectScan(this)
}

func (this *OrderedIntersectScan) New() Operator {
	return &OrderedIntersectScan{}
}

func (this *OrderedIntersectScan) Covers() expression.Covers {
	return this.scans[0].Covers()
}

func (this *OrderedIntersectScan) FilterCovers() map[*expression.Cover]value.Value {
	return this.scans[0].FilterCovers()
}

func (this *OrderedIntersectScan) Covering() bool {
	return this.scans[0].Covering()
}

func (this *OrderedIntersectScan) Scans() []SecondaryScan {
	return this.scans
}

func (this *OrderedIntersectScan) Limit() expression.Expression {
	return this.limit
}

func (this *OrderedIntersectScan) SetLimit(limit expression.Expression) {
	this.limit = limit
}

func (this *OrderedIntersectScan) Offset() expression.Expression {
	return nil
}

func (this *OrderedIntersectScan) SetOffset(limit expression.Expression) {
}

func (this *OrderedIntersectScan) GroupAggs() *IndexGroupAggregates {
	return nil
}

func (this *OrderedIntersectScan) OrderTerms() IndexKeyOrders {
	return this.scans[0].OrderTerms()
}

func (this *OrderedIntersectScan) SetCovers(covers expression.Covers) {
}

func (this *OrderedIntersectScan) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}

func (this *OrderedIntersectScan) CoverJoinSpanExpressions(coverer *expression.Coverer) error {
	for _, scan := range this.scans {
		err := scan.CoverJoinSpanExpressions(coverer)
		if err != nil {
			return err
		}
	}

	return nil
}

func (this *OrderedIntersectScan) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *OrderedIntersectScan) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "OrderedIntersectScan"}
	r["scans"] = this.scans

	if this.limit != nil {
		r["limit"] = expression.NewStringer().Visit(this.limit)
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *OrderedIntersectScan) UnmarshalJSON(body []byte) error {
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

func flattenOrderedIntersectScans(scan SecondaryScan, buf []SecondaryScan) []SecondaryScan {
	switch scan := scan.(type) {
	case *OrderedIntersectScan:
		buf = flattenOrderedIntersectScans(scan.scans[0], buf)
		buf = append(buf, scan.scans[1:]...)
	default:
		buf = append(buf, scan)
	}

	return buf
}
