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

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type Unnest struct {
	readonly
	optEstimate
	term   *algebra.Unnest
	alias  string
	filter expression.Expression
}

func NewUnnest(term *algebra.Unnest, filter expression.Expression, cost, cardinality float64) *Unnest {
	rv := &Unnest{
		term:   term,
		alias:  term.Alias(),
		filter: filter,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality)
	return rv
}

func (this *Unnest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnnest(this)
}

func (this *Unnest) New() Operator {
	return &Unnest{}
}

func (this *Unnest) Term() *algebra.Unnest {
	return this.term
}

func (this *Unnest) Alias() string {
	return this.alias
}

func (this *Unnest) Filter() expression.Expression {
	return this.filter
}

func (this *Unnest) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Unnest) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Unnest"}

	if this.term.Outer() {
		r["outer"] = this.term.Outer()
	}

	r["expr"] = expression.NewStringer().Visit(this.term.Expression())
	if this.alias != "" {
		r["as"] = this.alias
	}

	if this.filter != nil {
		r["filter"] = expression.NewStringer().Visit(this.filter)
	}

	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *Unnest) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string             `json:"#operator"`
		Outer       bool               `json:"outer"`
		Expr        string             `json:"expr"`
		As          string             `json:"as"`
		Filter      string             `json:"filter"`
		OptEstimate map[string]float64 `json:"optimizer_estimates"`
	}
	var expr expression.Expression

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	if _unmarshalled.Expr != "" {
		expr, err = parser.Parse(_unmarshalled.Expr)
		if err != nil {
			return err
		}
	}

	if _unmarshalled.Filter != "" {
		this.filter, err = parser.Parse(_unmarshalled.Filter)
		if err != nil {
			return err
		}
	}

	this.term = algebra.NewUnnest(nil, _unmarshalled.Outer, expr, _unmarshalled.As)
	this.alias = _unmarshalled.As

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

	return nil
}
