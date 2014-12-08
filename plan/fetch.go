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

	"github.com/couchbaselabs/query/algebra"
	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/expression/parser"
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
	var _fetch struct {
		Operator string `json:"#operator"`
		Proj     string `json:"projection"`
		Names    string `json:"namespace"`
		Keys     string `json:"keyspace"`
		As       string `json:"as"`
	}
	err := json.Unmarshal(body, &_fetch)

	if err != nil {
		return err
	}

	expr, err := parser.Parse(_fetch.Proj)
	if err != nil {
		return err
	}

	proj_expr, is_path := expr.(expression.Path)

	if !is_path {
		return fmt.Errorf("Fetch.UnmarshalJSON: cannot resolve path expression from %s", _fetch.Proj)
	}

	keys_expr, err := parser.Parse(_fetch.Keys)
	if err != nil {
		return err
	}

	this.term = algebra.NewKeyspaceTerm(_fetch.Names, _fetch.Keys,
		proj_expr, _fetch.As, keys_expr)

	namespace, err := datastore.GetDatastore().NamespaceByName(_fetch.Names)

	if err != nil {
		return err
	}

	this.keyspace, err = namespace.KeyspaceByName(_fetch.Keys)

	return err
}
