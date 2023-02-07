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
	"sort"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/value"
)

// IntersectScan that preserves index order of first scan.
type OrderedIntersectScan struct {
	readonly
	optEstimate
	scans   []SecondaryScan
	limit   expression.Expression
	allScan bool
}

func NewOrderedIntersectScan(limit expression.Expression, allScan bool, cost, cardinality float64,
	size int64, frCost float64, scans ...SecondaryScan) *OrderedIntersectScan {
	cbo := (cost > 0.0) && (cardinality > 0.0)
	for _, scan := range scans {
		if scan.Limit() != nil {
			scan.SetLimit(nil)
		}
		scan.SetOffset(nil)
		if cbo && (scan.Cost() <= 0.0 || scan.Cardinality() <= 0.0) {
			cbo = false
		}
	}

	buf := make([]SecondaryScan, 0, 2*len(scans))
	buf = flattenOrderedIntersectScans(scans[0], buf)
	scans = append(buf, flattenIntersectScans(scans[1:], buf[len(buf):])...)

	n := len(scans)
	if n > 64 {
		return NewOrderedIntersectScan(
			limit, allScan, cost, cardinality, size, frCost,
			scans[0],
			NewIntersectScan(nil, allScan, cost/2.0, cardinality, size, frCost, scans[1:n/2]...),
			NewIntersectScan(nil, allScan, cost/2.0, cardinality, size, frCost, scans[n/2:]...),
		)
	}

	if cbo {
		sort.Slice(scans, func(i, j int) bool {
			iCard := scans[i].Cardinality()
			jCard := scans[j].Cardinality()
			if iCard < jCard {
				return true
			} else if iCard > jCard {
				return false
			}
			return scans[i].Cost() <= scans[j].Cost()
		})
	}

	rv := &OrderedIntersectScan{
		scans:   scans,
		limit:   limit,
		allScan: allScan,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *OrderedIntersectScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitOrderedIntersectScan(this)
}

func (this *OrderedIntersectScan) New() Operator {
	return &OrderedIntersectScan{}
}

// AllScan means no early termination
func (this *OrderedIntersectScan) AllScan() bool {
	// if ordered intersect scan is chosen by cost, then do not do early termination
	return this.allScan || (this.cost > 0.0 && this.cardinality > 0.0 && this.size > 0 && this.frCost > 0.0)
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

func (this *OrderedIntersectScan) IsUnderNL() bool {
	return this.scans[0].IsUnderNL()
}

func (this *OrderedIntersectScan) CoverJoinSpanExpressions(coverer *expression.Coverer,
	implicitArrayKey *expression.All) error {
	for _, scan := range this.scans {
		err := scan.CoverJoinSpanExpressions(coverer, implicitArrayKey)
		if err != nil {
			return err
		}
	}

	return nil
}

func (this *OrderedIntersectScan) GetIndex() datastore.Index {
	return nil
}

func (this *OrderedIntersectScan) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *OrderedIntersectScan) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "OrderedIntersectScan"}

	if this.limit != nil {
		r["limit"] = expression.NewStringer().Visit(this.limit)
	}

	if this.allScan {
		r["all_scan"] = this.allScan
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

func (this *OrderedIntersectScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string                 `json:"#operator"`
		Scans       []json.RawMessage      `json:"scans"`
		Limit       string                 `json:"limit"`
		AllScan     bool                   `json:"all_scan"`
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

	this.allScan = _unmarshalled.AllScan

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

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

func (this *OrderedIntersectScan) verify(prepared *Prepared) bool {
	for _, scan := range this.scans {
		if !scan.verify(prepared) {
			return false
		}
	}

	return true
}
