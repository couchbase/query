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
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/value"
)

// Create index
type CreateIndex struct {
	readwrite
	keyspace datastore.Keyspace
	node     *algebra.CreateIndex
}

func NewCreateIndex(keyspace datastore.Keyspace, node *algebra.CreateIndex) *CreateIndex {
	return &CreateIndex{
		keyspace: keyspace,
		node:     node,
	}
}

func (this *CreateIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateIndex(this)
}

func (this *CreateIndex) New() Operator {
	return &CreateIndex{}
}

func (this *CreateIndex) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *CreateIndex) Node() *algebra.CreateIndex {
	return this.node
}

func (this *CreateIndex) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *CreateIndex) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "CreateIndex"}
	r["keyspace"] = this.keyspace.Name()
	r["namespace"] = this.keyspace.NamespaceId()
	r["index"] = this.node.Name()
	k := make([]interface{}, len(this.node.Keys()))
	for i, term := range this.node.Keys() {
		q := make(map[string]interface{}, 2)
		q["expr"] = term.Expression().String()

		if term.Descending() {
			q["desc"] = term.Descending()
		}

		k[i] = q
	}
	r["keys"] = k
	r["using"] = this.node.Using()

	if this.node.Partition() != nil {
		r["partition"] = this.node.Partition()
	}

	if this.node.Where() != nil {
		r["where"] = this.node.Where()
	}

	if this.node.With() != nil {
		r["with"] = this.node.With()
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *CreateIndex) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_      string `json:"#operator"`
		Keysp  string `json:"keyspace"`
		Namesp string `json:"namespace"`
		Index  string `json:"index"`
		Keys   []struct {
			Expr string `json:"expr"`
			Desc bool   `json:"desc"`
		} `json:"keys"`
		Using     datastore.IndexType `json:"using"`
		Partition []string            `json:"partition"`
		Where     string              `json:"where"`
		With      json.RawMessage     `json:"with"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Namesp, _unmarshalled.Keysp)
	if err != nil {
		return err
	}

	ksref := algebra.NewKeyspaceRef(_unmarshalled.Namesp, _unmarshalled.Keysp, "")

	var expr expression.Expression
	keys := make(algebra.IndexKeyTerms, len(_unmarshalled.Keys))

	for i, term := range _unmarshalled.Keys {
		expr, err = parser.Parse(term.Expr)
		if err != nil {
			return err
		}
		keys[i] = algebra.NewIndexKeyTerm(expr, term.Desc)
	}

	if keys.HasDescending() {
		indexer, err1 := this.keyspace.Indexer(_unmarshalled.Using)
		if err1 != nil {
			return err1
		}
		if _, ok := indexer.(datastore.Indexer2); !ok {
			return errors.NewIndexerDescCollationError()
		}
	}

	var partition expression.Expressions
	if len(_unmarshalled.Partition) > 0 {
		partition = make(expression.Expressions, len(_unmarshalled.Partition))
		for i, p := range _unmarshalled.Partition {
			partition[i], err = parser.Parse(p)
			if err != nil {
				return err
			}
		}
	}

	var where expression.Expression
	if _unmarshalled.Where != "" {
		where, err = parser.Parse(_unmarshalled.Where)
		if err != nil {
			return err
		}
	}

	var with value.Value
	if len(_unmarshalled.With) > 0 {
		with = value.NewValue(_unmarshalled.With)
	}

	this.node = algebra.NewCreateIndex(_unmarshalled.Index, ksref,
		keys, partition, where, _unmarshalled.Using, with)
	return nil
}
