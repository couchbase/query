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
)

type Stream struct {
	readonly
	serializable bool
	optEstimate
}

func NewStream(cost, cardinality float64, size int64, frCost float64, serializable bool) *Stream {
	rv := &Stream{}
	rv.serializable = serializable
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *Stream) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitStream(this)
}

func (this *Stream) New() Operator {
	return &Stream{}
}

func (this *Stream) Serializable() bool {
	return this.serializable
}

func (this *Stream) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Stream) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Stream"}
	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}

	r["serializable"] = this.serializable

	if f != nil {
		f(r)
	}
	return r
}

func (this *Stream) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_            string                 `json:"#operator"`
		OptEstimate  map[string]interface{} `json:"optimizer_estimates"`
		Serializable bool                   `json:"serializable"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

	this.serializable = _unmarshalled.Serializable

	return nil
}
