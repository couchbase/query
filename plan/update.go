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
	"github.com/couchbaselabs/query/expression"
)

// Enable copy-before-write, so that all reads use old values
type Clone struct {
	readonly
}

// Write to copy
type Set struct {
	readonly
	node *algebra.Set
}

// Write to copy
type Unset struct {
	readonly
	node *algebra.Unset
}

// Send to keyspace
type SendUpdate struct {
	readwrite
	keyspace datastore.Keyspace
	alias    string
}

func NewClone() *Clone {
	return &Clone{}
}

func (this *Clone) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitClone(this)
}

func (this *Clone) New() Operator {
	return &Clone{}
}

func (this *Clone) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "Clone"}
	return json.Marshal(r)
}

func (this *Clone) UnmarshalJSON([]byte) error {
	// TODO: Implement. Verify #operator is Clone, otherwise error
	return nil
}

func NewSet(node *algebra.Set) *Set {
	return &Set{
		node: node,
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

func (this *Set) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "Set"}
	s := make([]interface{}, 0, len(this.node.Terms()))
	for _, term := range this.node.Terms() {
		t := make(map[string]interface{})
		t["path"] = expression.NewStringer().Visit(term.Path())
		t["expr"] = expression.NewStringer().Visit(term.Value())
		s = append(s, t)
	}
	r["set_terms"] = s
	return json.Marshal(r)
}

func (this *Set) UnmarshalJSON([]byte) error {
	// TODO: Implement.
	return nil
}

func NewUnset(node *algebra.Unset) *Unset {
	return &Unset{
		node: node,
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

func (this *Unset) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "Unset"}
	s := make([]interface{}, 0, len(this.node.Terms()))
	for _, term := range this.node.Terms() {
		t := make(map[string]interface{})
		t["path"] = expression.NewStringer().Visit(term.Path())
		// FIXME
		//t["expr"] = expression.NewStringer().Visit(term.UpdateFor().Bindings())
		t["expr"] = "FIXME"
		s = append(s, t)
	}
	r["unset_terms"] = s

	return json.Marshal(r)
}

func (this *Unset) UnmarshalJSON([]byte) error {
	// TODO: Implement.
	return nil
}

func NewSendUpdate(keyspace datastore.Keyspace, alias string) *SendUpdate {
	return &SendUpdate{
		keyspace: keyspace,
		alias:    alias,
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

func (this *SendUpdate) Alias() string {
	return this.alias
}

func (this *SendUpdate) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "SendUpdate"}
	r["keyspace"] = this.keyspace.Name()
	r["alias"] = this.alias
	return json.Marshal(r)
}

func (this *SendUpdate) UnmarshalJSON([]byte) error {
	// TODO: Implement.
	return nil
}
