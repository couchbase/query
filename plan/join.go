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

func NewJoinFromAnsi(keyspace datastore.Keyspace, term *algebra.KeyspaceTerm, outer bool) *Join {
	return &Join{
		keyspace: keyspace,
		term:     term,
		outer:    outer,
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
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Join) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
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
	if f != nil {
		f(r)
	}
	return r
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
	this.term = algebra.NewKeyspaceTerm(_unmarshalled.Names, _unmarshalled.Keys, _unmarshalled.As, keys_expr, nil)
	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Names, _unmarshalled.Keys)
	return err
}
