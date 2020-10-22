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

type Fetch struct {
	readonly
	keyspace    datastore.Keyspace
	term        *algebra.KeyspaceTerm
	subPaths    []string
	cost        float64
	cardinality float64
}

func NewFetch(keyspace datastore.Keyspace, term *algebra.KeyspaceTerm, subPaths []string,
	cost, cardinality float64) *Fetch {
	return &Fetch{
		keyspace:    keyspace,
		term:        term,
		subPaths:    subPaths,
		cost:        cost,
		cardinality: cardinality,
	}
}

func (this *Fetch) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFetch(this)
}

func (this *Fetch) New() Operator {
	return &Fetch{}
}

func (this *Fetch) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *Fetch) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *Fetch) SubPaths() []string {
	return this.subPaths
}

func (this *Fetch) Cost() float64 {
	return this.cost
}

func (this *Fetch) Cardinality() float64 {
	return this.cardinality
}

func (this *Fetch) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Fetch) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Fetch"}
	this.term.MarshalKeyspace(r)
	if len(this.subPaths) > 0 {
		r["subpaths"] = this.subPaths
	}

	if this.term.As() != "" {
		r["as"] = this.term.As()
	}
	if this.term.IsUnderNL() {
		r["nested_loop"] = this.term.IsUnderNL()
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

func (this *Fetch) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string   `json:"#operator"`
		Namespace   string   `json:"namespace"`
		Bucket      string   `json:"bucket"`
		Scope       string   `json:"scope"`
		Keyspace    string   `json:"keyspace"`
		FromExpr    string   `json:"fromExpr"`
		As          string   `json:"as"`
		UnderNL     bool     `json:"nested_loop"`
		Cost        float64  `json:"cost"`
		Cardinality float64  `json:"cardinality"`
		SubPaths    []string `json:"subpaths"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.subPaths = _unmarshalled.SubPaths
	this.cost = getCost(_unmarshalled.Cost)
	this.cardinality = getCardinality(_unmarshalled.Cardinality)
	if _unmarshalled.FromExpr != "" {
		var expr expression.Expression
		expr, err = parser.Parse(_unmarshalled.FromExpr)
		if err == nil {
			this.term = algebra.NewKeyspaceTermFromExpression(expr, _unmarshalled.As, nil, nil, 0)
		}
	} else {
		this.term = algebra.NewKeyspaceTermFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
			_unmarshalled.Scope, _unmarshalled.Keyspace), _unmarshalled.As, nil, nil)
		this.keyspace, err = datastore.GetKeyspace(this.term.Path().Parts()...)
	}
	if err == nil && _unmarshalled.UnderNL {
		this.term.SetUnderNL()
	}

	return err
}

func (this *Fetch) verify(prepared *Prepared) bool {
	var res bool

	this.keyspace, res = verifyKeyspace(this.keyspace, prepared)
	return res
}

type DummyFetch struct {
	readonly
	keyspace    datastore.Keyspace
	term        *algebra.KeyspaceTerm
	cost        float64
	cardinality float64
}

func NewDummyFetch(keyspace datastore.Keyspace, term *algebra.KeyspaceTerm, cost, cardinality float64) *DummyFetch {
	return &DummyFetch{
		keyspace:    keyspace,
		term:        term,
		cost:        cost,
		cardinality: cardinality,
	}
}

func (this *DummyFetch) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDummyFetch(this)
}

func (this *DummyFetch) New() Operator {
	return &DummyFetch{}
}

func (this *DummyFetch) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *DummyFetch) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *DummyFetch) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *DummyFetch) Cost() float64 {
	return this.cost
}

func (this *DummyFetch) Cardinality() float64 {
	return this.cardinality
}

func (this *DummyFetch) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "DummyFetch"}
	this.term.MarshalKeyspace(r)
	if this.term.As() != "" {
		r["as"] = this.term.As()
	}
	if this.term.IsUnderNL() {
		r["nested_loop"] = this.term.IsUnderNL()
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

func (this *DummyFetch) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string  `json:"#operator"`
		Namespace   string  `json:"namespace"`
		Bucket      string  `json:"bucket"`
		Scope       string  `json:"scope"`
		Keyspace    string  `json:"keyspace"`
		FromExpr    string  `json:"fromExpr"`
		As          string  `json:"as"`
		UnderNL     bool    `json:"nested_loop"`
		Cost        float64 `json:"cost"`
		Cardinality float64 `json:"cardinality"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.cost = getCost(_unmarshalled.Cost)
	this.cardinality = getCardinality(_unmarshalled.Cardinality)
	if _unmarshalled.FromExpr != "" {
		expr, err1 := parser.Parse(_unmarshalled.FromExpr)
		if err1 != nil {
			return err1
		}
		this.term = algebra.NewKeyspaceTermFromExpression(expr, _unmarshalled.As, nil, nil, 0)
	} else {
		this.term = algebra.NewKeyspaceTermFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
			_unmarshalled.Scope, _unmarshalled.Keyspace), _unmarshalled.As, nil, nil)
		this.keyspace, err = datastore.GetKeyspace(this.term.Path().Parts()...)
	}

	if err == nil && _unmarshalled.UnderNL {
		this.term.SetUnderNL()
	}
	return err
}
