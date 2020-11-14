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
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type Join struct {
	readonly
	keyspace    datastore.Keyspace
	term        *algebra.KeyspaceTerm
	outer       bool
	onFilter    expression.Expression
	cost        float64
	cardinality float64
}

func NewJoin(keyspace datastore.Keyspace, join *algebra.Join, cost, cardinality float64) *Join {
	return &Join{
		keyspace:    keyspace,
		term:        join.Right(),
		outer:       join.Outer(),
		cost:        cost,
		cardinality: cardinality,
	}
}

func NewJoinFromAnsi(keyspace datastore.Keyspace, term *algebra.KeyspaceTerm, outer bool,
	onFilter expression.Expression, cost, cardinality float64) *Join {
	return &Join{
		keyspace:    keyspace,
		term:        term,
		outer:       outer,
		onFilter:    onFilter,
		cost:        cost,
		cardinality: cardinality,
	}
}

func (this *Join) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitJoin(this)
}

func (this *Join) New() Operator {
	return &Join{}
}

func (this *Join) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *Join) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *Join) Outer() bool {
	return this.outer
}

func (this *Join) OnFilter() expression.Expression {
	return this.onFilter
}

func (this *Join) Cost() float64 {
	return this.cost
}

func (this *Join) Cardinality() float64 {
	return this.cardinality
}

func (this *Join) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Join) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Join"}
	this.term.MarshalKeyspace(r)
	r["on_keys"] = expression.NewStringer().Visit(this.term.JoinKeys())

	if this.outer {
		r["outer"] = this.outer
	}

	if this.term.As() != "" {
		r["as"] = this.term.As()
	}

	if this.onFilter != nil {
		r["on_filter"] = expression.NewStringer().Visit(this.onFilter)
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

func (this *Join) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string  `json:"#operator"`
		Namespace   string  `json:"namespace"`
		Bucket      string  `json:"bucket"`
		Scope       string  `json:"scope"`
		Keyspace    string  `json:"keyspace"`
		On          string  `json:"on_keys"`
		Outer       bool    `json:"outer"`
		As          string  `json:"as"`
		OnFilter    string  `json:"on_filter"`
		Cost        float64 `json:"cost"`
		Cardinality float64 `json:"cardinality"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	var keys_expr expression.Expression
	if _unmarshalled.On != "" {
		keys_expr, err = parser.Parse(_unmarshalled.On)
		if err != nil {
			return err
		}
	}

	this.outer = _unmarshalled.Outer
	this.term = algebra.NewKeyspaceTermFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope, _unmarshalled.Keyspace), _unmarshalled.As, nil, nil)
	this.term.SetJoinKeys(keys_expr)
	this.keyspace, err = datastore.GetKeyspace(this.term.Path().Parts()...)
	if err != nil {
		return err
	}

	if _unmarshalled.OnFilter != "" {
		this.onFilter, err = parser.Parse(_unmarshalled.OnFilter)
		if err != nil {
			return err
		}
	}

	this.cost = getCost(_unmarshalled.Cost)
	this.cardinality = getCardinality(_unmarshalled.Cardinality)

	return nil
}

func (this *Join) verify(prepared *Prepared) bool {
	var res bool

	this.keyspace, res = verifyKeyspace(this.keyspace, prepared)
	return res
}
