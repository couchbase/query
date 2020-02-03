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
	this.node.Keyspace().MarshalKeyspace(r)
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

	if this.node.Partition() != nil && this.node.Partition().Strategy() != datastore.NO_PARTITION {
		q := make(map[string]interface{}, 2)
		q["exprs"] = this.node.Partition().Expressions()
		q["strategy"] = this.node.Partition().Strategy()
		r["partition"] = q
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
		_         string `json:"#operator"`
		Namespace string `json:"namespace"`
		Bucket    string `json:"bucket"`
		Scope     string `json:"scope"`
		Keyspace  string `json:"keyspace"`
		Index     string `json:"index"`
		Keys      []struct {
			Expr string `json:"expr"`
			Desc bool   `json:"desc"`
		} `json:"keys"`
		Using     datastore.IndexType `json:"using"`
		Partition *struct {
			Exprs    []string                `json:"exprs"`
			Strategy datastore.PartitionType `json:"strategy"`
		} `json:"partition"`
		Where string          `json:"where"`
		With  json.RawMessage `json:"with"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	ksref := algebra.NewKeyspaceRefFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope, _unmarshalled.Keyspace), "")
	this.keyspace, err = datastore.GetKeyspace(ksref.Path().Parts()...)
	if err != nil {
		return err
	}

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

	var partition *algebra.IndexPartitionTerm
	if _unmarshalled.Partition != nil {
		exprs := make(expression.Expressions, len(_unmarshalled.Partition.Exprs))
		for i, p := range _unmarshalled.Partition.Exprs {
			exprs[i], err = parser.Parse(p)
			if err != nil {
				return err
			}
		}
		partition = algebra.NewIndexPartitionTerm(_unmarshalled.Partition.Strategy, exprs)
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
		with = value.NewValue([]byte(_unmarshalled.With))
	}

	this.node = algebra.NewCreateIndex(_unmarshalled.Index, ksref,
		keys, partition, where, _unmarshalled.Using, with)
	return nil
}

func (this *CreateIndex) verify(prepared *Prepared) bool {
	var res bool

	this.keyspace, res = verifyKeyspace(this.keyspace, prepared)
	return res
}
