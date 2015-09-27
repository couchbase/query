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
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type Fetch struct {
	readonly
	keyspace datastore.Keyspace
	term     *algebra.KeyspaceTerm
}

func NewFetch(keyspace datastore.Keyspace, term *algebra.KeyspaceTerm) *Fetch {
	return &Fetch{
		keyspace: keyspace,
		term:     term,
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

func (this *Fetch) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "Fetch"}
	if this.term.Projection() != nil {
		r["projection"] = expression.NewStringer().Visit(this.term.Projection())
	}
	r["namespace"] = this.term.Namespace()
	r["keyspace"] = this.term.Keyspace()
	if this.term.As() != "" {
		r["as"] = this.term.As()
	}
	return json.Marshal(r)
}

func (this *Fetch) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string `json:"#operator"`
		Proj  string `json:"projection"`
		Names string `json:"namespace"`
		Keys  string `json:"keyspace"`
		As    string `json:"as"`
	}
	var proj_expr expression.Path

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	if _unmarshalled.Proj != "" {
		expr, err := parser.Parse(_unmarshalled.Proj)
		if err != nil {
			return err
		}

		_proj_expr, is_path := expr.(expression.Path)
		if !is_path {
			return fmt.Errorf("Fetch.UnmarshalJSON: cannot resolve path expression from %s", _unmarshalled.Proj)
		}
		proj_expr = _proj_expr
	}
	this.term = algebra.NewKeyspaceTerm(_unmarshalled.Names, _unmarshalled.Keys,
		proj_expr, _unmarshalled.As, nil, nil)

	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Names, _unmarshalled.Keys)

	return err
}

type DummyFetch struct {
	readonly
	keyspace datastore.Keyspace
	term     *algebra.KeyspaceTerm
}

func NewDummyFetch(keyspace datastore.Keyspace, term *algebra.KeyspaceTerm) *DummyFetch {
	return &DummyFetch{
		keyspace: keyspace,
		term:     term,
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
	r := map[string]interface{}{"#operator": "DummyFetch"}
	if this.term.Projection() != nil {
		r["projection"] = expression.NewStringer().Visit(this.term.Projection())
	}
	r["namespace"] = this.term.Namespace()
	r["keyspace"] = this.term.Keyspace()
	if this.term.As() != "" {
		r["as"] = this.term.As()
	}
	return json.Marshal(r)
}

func (this *DummyFetch) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string `json:"#operator"`
		Proj  string `json:"projection"`
		Names string `json:"namespace"`
		Keys  string `json:"keyspace"`
		As    string `json:"as"`
	}
	var proj_expr expression.Path

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	if _unmarshalled.Proj != "" {
		expr, err := parser.Parse(_unmarshalled.Proj)
		if err != nil {
			return err
		}

		_proj_expr, is_path := expr.(expression.Path)
		if !is_path {
			return fmt.Errorf("DummyFetch.UnmarshalJSON: cannot resolve path expression from %s", _unmarshalled.Proj)
		}
		proj_expr = _proj_expr
	}
	this.term = algebra.NewKeyspaceTerm(_unmarshalled.Names, _unmarshalled.Keys,
		proj_expr, _unmarshalled.As, nil, nil)

	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Names, _unmarshalled.Keys)

	return err
}
