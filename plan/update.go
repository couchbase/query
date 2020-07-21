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
	"github.com/couchbase/query/algebra/unmarshal"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

// Enable copy-before-write, so that all reads use old values
type Clone struct {
	readonly
	alias       string
	cost        float64
	cardinality float64
}

func NewClone(alias string, cost, cardinality float64) *Clone {
	return &Clone{
		alias:       alias,
		cost:        cost,
		cardinality: cardinality,
	}
}

func (this *Clone) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitClone(this)
}

func (this *Clone) New() Operator {
	return &Clone{}
}

func (this *Clone) Alias() string {
	return this.alias
}

func (this *Clone) Cost() float64 {
	return this.cost
}

func (this *Clone) Cardinality() float64 {
	return this.cardinality
}

func (this *Clone) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Clone) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Clone"}
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

func (this *Clone) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		Cost        float64 `json:"cost"`
		Cardinality float64 `json:"cardinality"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.cost = getCost(_unmarshalled.Cost)
	this.cardinality = getCardinality(_unmarshalled.Cardinality)

	return nil
}

// Write to copy
type Set struct {
	readonly
	node        *algebra.Set
	cost        float64
	cardinality float64
}

func NewSet(node *algebra.Set, cost, cardinality float64) *Set {
	return &Set{
		node:        node,
		cost:        cost,
		cardinality: cardinality,
	}
}

func (this *Set) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSet(this)
}

func (this *Set) New() Operator {
	return &Set{}
}

func (this *Set) Node() *algebra.Set {
	return this.node
}

func (this *Set) Cost() float64 {
	return this.cost
}

func (this *Set) Cardinality() float64 {
	return this.cardinality
}

func (this *Set) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Set) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Set"}
	r["set_terms"] = this.node.Terms()
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

func (this *Set) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string          `json:"#operator"`
		SetTerms    json.RawMessage `json:"set_terms"`
		Cost        float64         `json:"cost"`
		Cardinality float64         `json:"cardinality"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	terms, err := unmarshal.UnmarshalSetTerms(_unmarshalled.SetTerms)
	if err != nil {
		return err
	}

	this.node = algebra.NewSet(terms)

	this.cost = getCost(_unmarshalled.Cost)
	this.cardinality = getCardinality(_unmarshalled.Cardinality)

	return nil
}

// Write to copy
type Unset struct {
	readonly
	node        *algebra.Unset
	cost        float64
	cardinality float64
}

func NewUnset(node *algebra.Unset, cost, cardinality float64) *Unset {
	return &Unset{
		node:        node,
		cost:        cost,
		cardinality: cardinality,
	}
}

func (this *Unset) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnset(this)
}

func (this *Unset) New() Operator {
	return &Unset{}
}

func (this *Unset) Node() *algebra.Unset {
	return this.node
}

func (this *Unset) Cost() float64 {
	return this.cost
}

func (this *Unset) Cardinality() float64 {
	return this.cardinality
}

func (this *Unset) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Unset) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Unset"}
	r["unset_terms"] = this.node.Terms()
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

func (this *Unset) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string          `json:"#operator"`
		UnsetTerms  json.RawMessage `json:"unset_terms"`
		Cost        float64         `json:"cost"`
		Cardinality float64         `json:"cardinality"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	terms, err := unmarshal.UnmarshalUnsetTerms(_unmarshalled.UnsetTerms)
	if err != nil {
		return err
	}

	this.node = algebra.NewUnset(terms)

	this.cost = getCost(_unmarshalled.Cost)
	this.cardinality = getCardinality(_unmarshalled.Cardinality)

	return nil
}

// Send to keyspace
type SendUpdate struct {
	dml
	keyspace    datastore.Keyspace
	term        *algebra.KeyspaceRef
	alias       string
	limit       expression.Expression
	cost        float64
	cardinality float64
}

func NewSendUpdate(keyspace datastore.Keyspace, ksref *algebra.KeyspaceRef,
	limit expression.Expression, cost, cardinality float64) *SendUpdate {
	return &SendUpdate{
		keyspace:    keyspace,
		term:        ksref,
		alias:       ksref.Alias(),
		limit:       limit,
		cost:        cost,
		cardinality: cardinality,
	}
}

func (this *SendUpdate) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSendUpdate(this)
}

func (this *SendUpdate) New() Operator {
	return &SendUpdate{}
}

func (this *SendUpdate) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *SendUpdate) Term() *algebra.KeyspaceRef {
	return this.term
}

func (this *SendUpdate) Alias() string {
	return this.alias
}

func (this *SendUpdate) Limit() expression.Expression {
	return this.limit
}

func (this *SendUpdate) Cost() float64 {
	return this.cost
}

func (this *SendUpdate) Cardinality() float64 {
	return this.cardinality
}

func (this *SendUpdate) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *SendUpdate) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "SendUpdate"}
	this.term.MarshalKeyspace(r)
	r["alias"] = this.alias

	if this.limit != nil {
		r["limit"] = this.limit
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

func (this *SendUpdate) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string  `json:"#operator"`
		Namespace   string  `json:"namespace"`
		Bucket      string  `json:"bucket"`
		Scope       string  `json:"scope"`
		Keyspace    string  `json:"keyspace"`
		Expr        string  `json:"expr"`
		As          string  `json:"as"`
		Alias       string  `json:"alias"`
		Limit       string  `json:"limit"`
		Cost        float64 `json:"cost"`
		Cardinality float64 `json:"cardinality"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.alias = _unmarshalled.Alias

	if _unmarshalled.Limit != "" {
		this.limit, err = parser.Parse(_unmarshalled.Limit)
		if err != nil {
			return err
		}
	}

	this.cost = getCost(_unmarshalled.Cost)
	this.cardinality = getCardinality(_unmarshalled.Cardinality)

	if _unmarshalled.Expr != "" {
		var expr expression.Expression
		expr, err = parser.Parse(_unmarshalled.Expr)
		if err == nil {
			this.term = algebra.NewKeyspaceRefFromExpression(expr, _unmarshalled.As)
		}
	} else {
		this.term = algebra.NewKeyspaceRefFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
			_unmarshalled.Scope, _unmarshalled.Keyspace), _unmarshalled.As)
		this.keyspace, err = datastore.GetKeyspace(this.term.Path().Parts()...)
	}

	return err
}

func (this *SendUpdate) verify(prepared *Prepared) bool {
	var res bool

	this.keyspace, res = verifyKeyspace(this.keyspace, prepared)
	return res
}
