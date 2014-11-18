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

func (this *Join) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *Join) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *Join) Outer() bool {
	return this.outer
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

func (this *Nest) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *Nest) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *Nest) Outer() bool {
	return this.outer
}

/*
func (this *Nest) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "nest"}
	r["keyspace"] = this.keyspace.Name()
	r["term"] = this.term
	r["as"] = this.alias
	return json.Marshal(r)
}
*/

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
