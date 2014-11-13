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
	"github.com/couchbaselabs/query/algebra"
	"github.com/couchbaselabs/query/datastore"
)

type Join struct {
	readonly
	keyspace datastore.Keyspace
	term     *algebra.Join
	alias    string
}

func NewJoin(keyspace datastore.Keyspace, term *algebra.Join) *Join {
	return &Join{
		keyspace: keyspace,
		term:     term,
		alias:    term.Alias(),
	}
}

func (this *Join) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitJoin(this)
}

func (this *Join) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *Join) Term() *algebra.Join {
	return this.term
}

func (this *Join) Alias() string {
	return this.alias
}

type Nest struct {
	readonly
	keyspace datastore.Keyspace
	term     *algebra.Nest
	alias    string
}

func NewNest(keyspace datastore.Keyspace, term *algebra.Nest) *Nest {
	return &Nest{
		keyspace: keyspace,
		term:     term,
		alias:    term.Alias(),
	}
}

func (this *Nest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitNest(this)
}

func (this *Nest) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *Nest) Term() *algebra.Nest {
	return this.term
}

func (this *Nest) Alias() string {
	return this.alias
}

func (this *Nest) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "nest"}
	r["keyspace"] = this.keyspace.Name()
	r["term"] = this.term
	r["as"] = this.alias
	return json.Marshal(r)
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

func (this *Unnest) Term() *algebra.Unnest {
	return this.term
}

func (this *Unnest) Alias() string {
	return this.alias
}

func (this *Unnest) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "unnest"}
	r["term"] = this.term
	r["as"] = this.alias
	return json.Marshal(r)
}
