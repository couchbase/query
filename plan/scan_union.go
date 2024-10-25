//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import (
	"encoding/json"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

// UnionScan scans multiple indexes and unions the results.
type UnionScan struct {
	readonly
	optEstimate
	scans  []SecondaryScan
	limit  expression.Expression
	offset expression.Expression
}

func NewUnionScan(limit, offset expression.Expression, cost, cardinality float64,
	size int64, frCost float64, scans ...SecondaryScan) *UnionScan {
	for _, scan := range scans {
		if scan.Limit() == nil {
			limit = nil
			offset = nil
			break
		}
	}

	rv := &UnionScan{
		scans:  scans,
		limit:  limit,
		offset: offset,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
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

func (this *UnionScan) IsUnderNL() bool {
	return this.scans[0].IsUnderNL()
}

func (this *UnionScan) CoverJoinSpanExpressions(coverer *expression.Coverer,
	implicitArrayKey *expression.All) error {
	for _, scan := range this.scans {
		err := scan.CoverJoinSpanExpressions(coverer, implicitArrayKey)
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

	for _, scan := range this.scans {
		found := false
		for _, s := range scans {
			if scan.Equals(s) {
				found = true
			}
		}
		if !found {
			scans = append(scans, scan)
		}
	}

	switch len(scans) {
	case 1:
		return scans[0]
	case len(this.scans):
		return this
	default:
		scan := NewUnionScan(this.limit, this.offset, this.cost, this.cardinality, this.size, this.frCost, scans...)
		this.limit = scan.Limit()
		this.offset = scan.Offset()
		return scan
	}
}

func (this *UnionScan) GetIndex() datastore.Index {
	var index datastore.Index
	for _, child := range this.scans {
		idx := child.GetIndex()
		if idx == nil {
			return nil
		}
		if index == nil {
			index = idx
		} else if idx != index {
			return nil
		}
	}
	return index
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

	if this.limit != nil {
		r["limit"] = this.limit.String()
	}

	if this.offset != nil {
		r["offset"] = this.offset.String()
	}

	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}

	if f != nil {
		f(r)
	} else {
		r["scans"] = this.scans
	}
	return r
}

func (this *UnionScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string                 `json:"#operator"`
		Scans       []json.RawMessage      `json:"scans"`
		Limit       string                 `json:"limit"`
		Offset      string                 `json:"offset"`
		OptEstimate map[string]interface{} `json:"optimizer_estimates"`
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

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

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

func (this *UnionScan) Equals(i interface{}) bool {
	if us, ok := i.(*UnionScan); ok {
		if len(this.scans) != len(us.scans) {
			return false
		}
		for _, s := range this.scans {
			found := false
			for _, ss := range us.scans {
				if s.Equals(ss) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		if this.limit != nil && us.limit != nil {
			if !this.limit.EquivalentTo(us.limit) {
				return false
			}
		} else if this.limit != us.limit {
			return false
		}
		if this.offset != nil && us.offset != nil {
			if !this.offset.EquivalentTo(us.offset) {
				return false
			}
		} else if this.offset != us.offset {
			return false
		}
		return true
	}
	return false
}

var _STRING_SCANS_POOL = util.NewStringBoolPool(16)
