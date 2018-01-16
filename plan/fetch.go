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
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Fetch) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Fetch"}
	r["namespace"] = this.term.Namespace()
	r["keyspace"] = this.term.Keyspace()
	if this.term.As() != "" {
		r["as"] = this.term.As()
	}
	if this.term.IsAnsiJoin() {
		r["ansi_join"] = this.term.IsAnsiJoin()
	}
	if this.term.IsAnsiNest() {
		r["ansi_nest"] = this.term.IsAnsiNest()
	}
	if f != nil {
		f(r)
	}
	return r
}

func (this *Fetch) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_        string `json:"#operator"`
		Names    string `json:"namespace"`
		Keys     string `json:"keyspace"`
		As       string `json:"as"`
		AnsiJoin bool   `json:"ansi_join"`
		AnsiNest bool   `json:"ansi_nest"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.term = algebra.NewKeyspaceTerm(_unmarshalled.Names, _unmarshalled.Keys, _unmarshalled.As, nil, nil)
	if _unmarshalled.AnsiJoin {
		this.term.SetAnsiJoin()
	}
	if _unmarshalled.AnsiNest {
		this.term.SetAnsiNest()
	}
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
	return json.Marshal(this.MarshalBase(nil))
}

func (this *DummyFetch) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "DummyFetch"}
	r["namespace"] = this.term.Namespace()
	r["keyspace"] = this.term.Keyspace()
	if this.term.As() != "" {
		r["as"] = this.term.As()
	}
	if this.term.IsAnsiJoin() {
		r["ansi_join"] = this.term.IsAnsiJoin()
	}
	if this.term.IsAnsiNest() {
		r["ansi_nest"] = this.term.IsAnsiNest()
	}
	if f != nil {
		f(r)
	}
	return r
}

func (this *DummyFetch) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_        string `json:"#operator"`
		Names    string `json:"namespace"`
		Keys     string `json:"keyspace"`
		As       string `json:"as"`
		AnsiJoin bool   `json:"ansi_join"`
		AnsiNest bool   `json:"ansi_nest"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.term = algebra.NewKeyspaceTerm(_unmarshalled.Names, _unmarshalled.Keys, _unmarshalled.As, nil, nil)
	if _unmarshalled.AnsiJoin {
		this.term.SetAnsiJoin()
	}
	if _unmarshalled.AnsiNest {
		this.term.SetAnsiNest()
	}
	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Names, _unmarshalled.Keys)
	return err
}

func (this *Fetch) verify(prepared *Prepared) bool {
	return verifyKeyspace(this.keyspace, prepared)
}
