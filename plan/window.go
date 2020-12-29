//  Copyright (c) 2018 Couchbase, Inc.
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

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type WindowAggregate struct {
	readonly
	optEstimate
	aggregates algebra.Aggregates
}

func NewWindowAggregate(aggregates algebra.Aggregates, cost, cardinality float64) *WindowAggregate {
	rv := &WindowAggregate{
		aggregates: aggregates,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality)
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
		s = append(s, expression.NewStringer().Visit(agg))
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
		_           string             `json:"#operator"`
		Aggs        []string           `json:"aggregates"`
		OptEstimate map[string]float64 `json:"optimizer_estimates"`
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
