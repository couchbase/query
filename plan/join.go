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
	keyspace datastore.Keyspace
	term     *algebra.KeyspaceTerm
	outer    bool
}

func NewJoin(keyspace datastore.Keyspace, join *algebra.Join) *Join {
	return &Join{
		keyspace: keyspace,
		term:     join.Right(),
		outer:    join.Outer(),
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

func (this *Join) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "Join"}
	r["namespace"] = this.term.Namespace()
	r["keyspace"] = this.term.Keyspace()
	r["on_keys"] = expression.NewStringer().Visit(this.term.Keys())

	if this.outer {
		r["outer"] = this.outer
	}

	if this.term.As() != "" {
		r["as"] = this.term.As()
	}
	return json.Marshal(r)
}

func (this *Join) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string `json:"#operator"`
		Names string `json:"namespace"`
		Keys  string `json:"keyspace"`
		On    string `json:"on_keys"`
		Outer bool   `json:"outer"`
		As    string `json:"as"`
	}
	var keys_expr expression.Expression

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	if _unmarshalled.On != "" {
		keys_expr, err = parser.Parse(_unmarshalled.On)
		if err != nil {
			return err
		}
	}

	this.outer = _unmarshalled.Outer
	this.term = algebra.NewKeyspaceTerm(_unmarshalled.Names, _unmarshalled.Keys,
		nil, _unmarshalled.As, keys_expr, nil)
	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Names, _unmarshalled.Keys)
	return err
}

type Nest struct {
	readonly
	keyspace datastore.Keyspace
	term     *algebra.KeyspaceTerm
	outer    bool
}

func NewNest(keyspace datastore.Keyspace, nest *algebra.Nest) *Nest {
	return &Nest{
		keyspace: keyspace,
		term:     nest.Right(),
		outer:    nest.Outer(),
	}
}

func (this *Nest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitNest(this)
}

func (this *Nest) New() Operator {
	return &Nest{}
}

func (this *Nest) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *Nest) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *Nest) Outer() bool {
	return this.outer
}

func (this *Nest) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "Nest"}
	r["namespace"] = this.term.Namespace()
	r["keyspace"] = this.term.Keyspace()
	r["on_keys"] = expression.NewStringer().Visit(this.term.Keys())

	if this.outer {
		r["outer"] = this.outer
	}

	if this.term.As() != "" {
		r["as"] = this.term.As()
	}
	return json.Marshal(r)
}

func (this *Nest) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string `json:"#operator"`
		Names string `json:"namespace"`
		Keys  string `json:"keyspace"`
		On    string `json:"on_keys"`
		Outer bool   `json:"outer"`
		As    string `json:"as"`
	}
	var keys_expr expression.Expression

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	if _unmarshalled.On != "" {
		keys_expr, err = parser.Parse(_unmarshalled.On)
		if err != nil {
			return err
		}
	}

	this.outer = _unmarshalled.Outer
	this.term = algebra.NewKeyspaceTerm(_unmarshalled.Names, _unmarshalled.Keys,
		nil, _unmarshalled.As, keys_expr, nil)
	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Names, _unmarshalled.Keys)
	return err

}

type Unnest struct {
	readonly
	term  *algebra.Unnest
	alias string
}

func NewUnnest(term *algebra.Unnest) *Unnest {
	return &Unnest{
		term:  term,
		alias: term.Alias(),
	}
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

func (this *Unnest) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "Unnest"}

	if this.term.Outer() {
		r["outer"] = this.term.Outer()
	}

	r["expr"] = expression.NewStringer().Visit(this.term.Expression())
	if this.alias != "" {
		r["as"] = this.alias
	}
	return json.Marshal(r)
}

func (this *Unnest) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string `json:"#operator"`
		Outer bool   `json:"outer"`
		Expr  string `json:"expr"`
		As    string `json:"as"`
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

	this.term = algebra.NewUnnest(nil, _unmarshalled.Outer, expr, _unmarshalled.As)
	this.alias = _unmarshalled.As
	return err
}
