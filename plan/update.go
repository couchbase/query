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

// Enable copy-before-write, so that all reads use old values
type Clone struct {
	readonly
	alias string
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
	limit    expression.Expression
}

func NewClone(alias string) *Clone {
	return &Clone{
		alias: alias,
	}
}

func (this *Clone) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitClone(this)
}

func (this *Clone) New() Operator {
	return &Clone{}
}

func (this *Clone) Alias() string {
	return this.alias
}

func (this *Clone) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "Clone"}
	return json.Marshal(r)
}

func (this *Clone) UnmarshalJSON([]byte) error {
	// NOP: Clone has no data structure
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

func (this *Set) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_        string `json:"#operator"`
		SetTerms []struct {
			Path string `json:"path"`
			Expr string `json:"expr"`
		} `json:"set_terms"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	terms := make([]*algebra.SetTerm, len(_unmarshalled.SetTerms))
	for i, SetTerm := range _unmarshalled.SetTerms {
		path_expr, err := parser.Parse(SetTerm.Path)
		if err != nil {
			return err
		}

		path, is_path := path_expr.(expression.Path)
		if !is_path {
			return fmt.Errorf("Set.UnmarshalJSON: cannot resolve path expression from %s", SetTerm.Path)
		}

		expr, err := parser.Parse(SetTerm.Expr)
		if err != nil {
			return err
		}

		terms[i] = algebra.NewSetTerm(path, expr, nil)
	}
	this.node = algebra.NewSet(terms)
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

func (this *Unset) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_          string `json:"#operator"`
		UnsetTerms []struct {
			Path string `json:"path"`
			Expr string `json:"expr"`
		} `json:"unset_terms"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	terms := make([]*algebra.UnsetTerm, len(_unmarshalled.UnsetTerms))
	for i, UnsetTerm := range _unmarshalled.UnsetTerms {
		path_expr, err := parser.Parse(UnsetTerm.Path)
		if err != nil {
			return err
		}

		path, is_path := path_expr.(expression.Path)
		if !is_path {
			return fmt.Errorf("Unset.UnmarshalJSON: cannot resolve path expression from %s", UnsetTerm.Path)
		}

		// is expr needed in Unset?
		_, err = parser.Parse(UnsetTerm.Expr)
		if err != nil {
			return err
		}

		terms[i] = algebra.NewUnsetTerm(path, nil)
	}
	this.node = algebra.NewUnset(terms)
	return nil
}

func NewSendUpdate(keyspace datastore.Keyspace, alias string, limit expression.Expression) *SendUpdate {
	return &SendUpdate{
		keyspace: keyspace,
		alias:    alias,
		limit:    limit,
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

func (this *SendUpdate) Limit() expression.Expression {
	return this.limit
}

func (this *SendUpdate) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "SendUpdate"}
	r["keyspace"] = this.keyspace.Name()
	r["namespace"] = this.keyspace.NamespaceId()
	r["alias"] = this.alias
	r["limit"] = this.limit
	return json.Marshal(r)
}

func (this *SendUpdate) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string `json:"#operator"`
		Keys  string `json:"keyspace"`
		Names string `json:"namespace"`
		Alias string `json:"alias"`
		Limit string `json:"limit"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.alias = _unmarshalled.Alias

	if _unmarshalled.Limit != "" {
		this.limit, err = parser.Parse(_unmarshalled.Limit)
		if err != nil {
			return err
		}
	}

	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Names, _unmarshalled.Keys)
	return err
}
