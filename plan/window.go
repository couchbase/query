//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import (
	"encoding/json"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/expression/parser"
)

type WindowAggregate struct {
	readonly
	optEstimate
	aggregates algebra.Aggregates
}

func NewWindowAggregate(aggregates algebra.Aggregates, cost, cardinality float64,
	size int64, frCost float64) *WindowAggregate {
	rv := &WindowAggregate{
		aggregates: aggregates,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *WindowAggregate) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitWindowAggregate(this)
}

func (this *WindowAggregate) New() Operator {
	return &WindowAggregate{}
}

func (this *WindowAggregate) Aggregates() algebra.Aggregates {
	return this.aggregates
}

func (this *WindowAggregate) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *WindowAggregate) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "WindowAggregate"}
	s := make([]interface{}, 0, len(this.aggregates))
	for _, agg := range this.aggregates {
		s = append(s, agg.String())
	}
	r["aggregates"] = s
	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}
	if f != nil {
		f(r)
	}
	return r
}

func (this *WindowAggregate) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string                 `json:"#operator"`
		Aggs        []string               `json:"aggregates"`
		OptEstimate map[string]interface{} `json:"optimizer_estimates"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.aggregates = make(algebra.Aggregates, len(_unmarshalled.Aggs))
	for i, agg := range _unmarshalled.Aggs {
		agg_expr, err := parser.Parse(agg)
		if err != nil {
			return err
		}
		this.aggregates[i], _ = agg_expr.(algebra.Aggregate)
	}

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

	return nil
}
