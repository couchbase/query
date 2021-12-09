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

type Nest struct {
	readonly
	keyspace datastore.Keyspace
	term     *algebra.KeyspaceTerm
	outer    bool
	onFilter expression.Expression
}

func NewNest(keyspace datastore.Keyspace, nest *algebra.Nest) *Nest {
	return &Nest{
		keyspace: keyspace,
		term:     nest.Right(),
		outer:    nest.Outer(),
	}
}

func NewNestFromAnsi(keyspace datastore.Keyspace, term *algebra.KeyspaceTerm, outer bool,
	onFilter expression.Expression) *Nest {
	return &Nest{
		keyspace: keyspace,
		term:     term,
		outer:    outer,
		onFilter: onFilter,
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

func (this *Nest) OnFilter() expression.Expression {
	return this.onFilter
}

func (this *Nest) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Nest) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Nest"}
	r["namespace"] = this.term.Namespace()
	r["keyspace"] = this.term.Keyspace()
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

	if f != nil {
		f(r)
	}
	return r
}

func (this *Nest) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_        string `json:"#operator"`
		Names    string `json:"namespace"`
		Keys     string `json:"keyspace"`
		On       string `json:"on_keys"`
		Outer    bool   `json:"outer"`
		As       string `json:"as"`
		OnFilter string `json:"on_filter"`
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
	this.term = algebra.NewKeyspaceTerm(_unmarshalled.Names, _unmarshalled.Keys, _unmarshalled.As, nil, nil)
	this.term.SetJoinKeys(keys_expr)
	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Names, _unmarshalled.Keys)
	if err != nil {
		return err
	}

	if _unmarshalled.OnFilter != "" {
		this.onFilter, err = parser.Parse(_unmarshalled.OnFilter)
		if err != nil {
			return err
		}
	}

	return nil
}

func (this *Nest) verify(prepared *Prepared) bool {
	var res bool

	this.keyspace, res = verifyKeyspace(this.keyspace, prepared)
	return res
}
