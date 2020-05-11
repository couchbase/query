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

type SendInsert struct {
	readwrite
	keyspace    datastore.Keyspace
	term        *algebra.KeyspaceRef
	alias       string
	key         expression.Expression
	value       expression.Expression
	options     expression.Expression
	limit       expression.Expression
	cost        float64
	cardinality float64
}

func NewSendInsert(keyspace datastore.Keyspace, ksref *algebra.KeyspaceRef,
	key, value, options, limit expression.Expression, cost, cardinality float64) *SendInsert {
	return &SendInsert{
		keyspace:    keyspace,
		term:        ksref,
		alias:       ksref.Alias(),
		key:         key,
		value:       value,
		options:     options,
		limit:       limit,
		cost:        cost,
		cardinality: cardinality,
	}
}

func (this *SendInsert) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSendInsert(this)
}

func (this *SendInsert) New() Operator {
	return &SendInsert{}
}

func (this *SendInsert) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *SendInsert) Alias() string {
	return this.alias
}

func (this *SendInsert) Key() expression.Expression {
	return this.key
}

func (this *SendInsert) Value() expression.Expression {
	return this.value
}

func (this *SendInsert) Options() expression.Expression {
	return this.options
}

func (this *SendInsert) Limit() expression.Expression {
	return this.limit
}

func (this *SendInsert) Cost() float64 {
	return this.cost
}

func (this *SendInsert) Cardinality() float64 {
	return this.cardinality
}

func (this *SendInsert) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *SendInsert) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "SendInsert"}
	this.term.MarshalKeyspace(r)
	r["alias"] = this.alias

	if this.limit != nil {
		r["limit"] = this.limit
	}

	if this.key != nil {
		r["key"] = this.key.String()
	}

	if this.value != nil {
		r["value"] = this.value.String()
	}

	if this.options != nil {
		r["options"] = this.options.String()
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

func (this *SendInsert) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string  `json:"#operator"`
		KeyExpr     string  `json:"key"`
		ValueExpr   string  `json:"value"`
		OptionsExpr string  `json:"options"`
		Namespace   string  `json:"namespace"`
		Bucket      string  `json:"bucket"`
		Scope       string  `json:"scope"`
		Keyspace    string  `json:"keyspace"`
		Alias       string  `json:"alias"`
		Limit       string  `json:"limit"`
		Cost        float64 `json:"cost"`
		Cardinality float64 `json:"cardinality"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	if _unmarshalled.KeyExpr != "" {
		this.key, err = parser.Parse(_unmarshalled.KeyExpr)
		if err != nil {
			return err
		}
	}

	if _unmarshalled.ValueExpr != "" {
		this.value, err = parser.Parse(_unmarshalled.ValueExpr)
		if err != nil {
			return err
		}
	}

	if _unmarshalled.OptionsExpr != "" {
		this.options, err = parser.Parse(_unmarshalled.OptionsExpr)
		if err != nil {
			return err
		}
	}

	this.alias = _unmarshalled.Alias

	if _unmarshalled.Limit != "" {
		this.limit, err = parser.Parse(_unmarshalled.Limit)
		if err != nil {
			return err
		}
	}

	this.term = algebra.NewKeyspaceRefFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope, _unmarshalled.Keyspace), "")
	this.keyspace, err = datastore.GetKeyspace(this.term.Path().Parts()...)
	if err != nil {
		return err
	}

	this.cost = getCost(_unmarshalled.Cost)
	this.cardinality = getCardinality(_unmarshalled.Cardinality)

	return nil
}

func (this *SendInsert) verify(prepared *Prepared) bool {
	var res bool

	this.keyspace, res = verifyKeyspace(this.keyspace, prepared)
	return res
}
