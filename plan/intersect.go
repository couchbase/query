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

	"github.com/couchbase/query/errors"
)

type IntersectAll struct {
	readonly
	optEstimate
	first    Operator
	second   Operator
	distinct bool
}

func NewIntersectAll(first, second Operator, distinct bool, cost, cardinality float64,
	size int64, frCost float64) *IntersectAll {
	rv := &IntersectAll{
		first:    first,
		second:   second,
		distinct: distinct,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *IntersectAll) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIntersectAll(this)
}

func (this *IntersectAll) New() Operator {
	return &IntersectAll{}
}

func (this *IntersectAll) First() Operator {
	return this.first
}

func (this *IntersectAll) Second() Operator {
	return this.second
}

func (this *IntersectAll) Distinct() bool {
	return this.distinct
}

func (this *IntersectAll) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *IntersectAll) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "IntersectAll"}
	if this.distinct {
		r["distinct"] = this.distinct
	}
	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}
	if f != nil {
		f(r)
	} else {
		r["first"] = this.first
		r["second"] = this.second
	}
	return r
}

func (this *IntersectAll) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string                 `json:"#operator"`
		First       json.RawMessage        `json:"first"`
		Second      json.RawMessage        `json:"second"`
		Distinct    bool                   `json:"distinct"`
		OptEstimate map[string]interface{} `json:"optimizer_estimates"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	planContext := this.PlanContext()

	for i, child := range []json.RawMessage{_unmarshalled.First, _unmarshalled.Second} {
		var op_type struct {
			Operator string `json:"#operator"`
		}

		err = json.Unmarshal(child, &op_type)
		if err != nil {
			return err
		}

		if i == 0 {
			this.first, err = MakeOperator(op_type.Operator, child, planContext)
		} else {
			this.second, err = MakeOperator(op_type.Operator, child, planContext)
		}

		if err != nil {
			return err
		}
	}

	if _unmarshalled.Distinct {
		this.distinct = true
	}

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

	return err
}

func (this *IntersectAll) verify(prepared *Prepared) errors.Error {
	err := this.first.verify(prepared)
	if err != nil {
		return err
	}
	return this.second.verify(prepared)
}

func (this *IntersectAll) keyspaceReferences(prepared *Prepared) {
	this.first.keyspaceReferences(prepared)
	this.second.keyspaceReferences(prepared)
}
