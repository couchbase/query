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
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/value"
)

// DistinctScan scans multiple indexes and distincts the results.
type DistinctScan struct {
	readonly
	optEstimate
	scan   SecondaryScan
	limit  expression.Expression
	offset expression.Expression
}

func NewDistinctScan(limit, offset expression.Expression, scan SecondaryScan, cost, cardinality float64,
	size int64, frCost float64) *DistinctScan {
	rv := &DistinctScan{
		scan:   scan,
		limit:  limit,
		offset: offset,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *DistinctScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDistinctScan(this)
}

func (this *DistinctScan) New() Operator {
	return &DistinctScan{}
}

func (this *DistinctScan) Covers() expression.Covers {
	return this.scan.Covers()
}

func (this *DistinctScan) SetCovers(covers expression.Covers) {
	this.scan.SetCovers(covers)
}

func (this *DistinctScan) FilterCovers() map[*expression.Cover]value.Value {
	return this.scan.FilterCovers()
}

func (this *DistinctScan) Covering() bool {
	return this.scan.Covering()
}

func (this *DistinctScan) Scan() SecondaryScan {
	return this.scan
}

func (this *DistinctScan) Limit() expression.Expression {
	return this.limit
}

func (this *DistinctScan) Offset() expression.Expression {
	return this.offset
}

func (this *DistinctScan) SetLimit(limit expression.Expression) {
	this.limit = limit
	this.scan.SetLimit(limit)
}

func (this *DistinctScan) SetOffset(offset expression.Expression) {
	this.offset = offset
	this.scan.SetOffset(offset)
}

func (this *DistinctScan) IsUnderNL() bool {
	return this.scan.IsUnderNL()
}

func (this *DistinctScan) CoverJoinSpanExpressions(coverer *expression.Coverer,
	implicitArrayKey *expression.All) error {
	return this.scan.CoverJoinSpanExpressions(coverer, implicitArrayKey)
}

func (this *DistinctScan) GroupAggs() *IndexGroupAggregates {
	return nil
}

func (this *DistinctScan) OrderTerms() IndexKeyOrders {
	return nil
}

func (this *DistinctScan) GetIndex() datastore.Index {
	return this.scan.GetIndex()
}

func (this *DistinctScan) SetImplicitArrayKey(arrayKey *expression.All) {
	this.scan.SetImplicitArrayKey(arrayKey)
}

func (this *DistinctScan) ImplicitArrayKey() *expression.All {
	return this.scan.ImplicitArrayKey()
}

func (this *DistinctScan) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}

func (this *DistinctScan) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *DistinctScan) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "DistinctScan"}

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
		r["scan"] = this.scan
	}
	return r
}

func (this *DistinctScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string                 `json:"#operator"`
		Scan        json.RawMessage        `json:"scan"`
		Limit       string                 `json:"limit"`
		Offset      string                 `json:"offset"`
		OptEstimate map[string]interface{} `json:"optimizer_estimates"`
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

	planContext := this.PlanContext()

	scan_op, err := MakeOperator(scan_type.Operator, _unmarshalled.Scan, planContext)
	if err != nil {
		return err
	}

	this.scan = scan_op.(SecondaryScan)

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

	if planContext != nil {
		if this.limit != nil {
			_, err = planContext.Map(this.limit)
			if err != nil {
				return err
			}
		}
		if this.offset != nil {
			_, err = planContext.Map(this.offset)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (this *DistinctScan) verify(prepared *Prepared) errors.Error {
	return this.scan.verify(prepared)
}

func (this *DistinctScan) keyspaceReferences(prepared *Prepared) {
	this.scan.keyspaceReferences(prepared)
}

func (this *DistinctScan) Equals(i interface{}) bool {
	if ds, ok := i.(*DistinctScan); ok {
		return this.String() == ds.String()
	}
	return false
}
