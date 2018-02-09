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

type IndexNest struct {
	readonly
	keyspace datastore.Keyspace
	term     *algebra.KeyspaceTerm
	outer    bool
	keyFor   string
	idExpr   expression.Expression
	index    datastore.Index
	indexer  datastore.Indexer
}

func NewIndexNest(keyspace datastore.Keyspace, nest *algebra.IndexNest,
	index datastore.Index) *IndexNest {
	rv := &IndexNest{
		keyspace: keyspace,
		term:     nest.Right(),
		outer:    nest.Outer(),
		keyFor:   nest.For(),
		index:    index,
		indexer:  index.Indexer(),
	}

	rv.idExpr = expression.NewField(
		expression.NewMeta(expression.NewIdentifier(rv.keyFor)),
		expression.NewFieldName("id", false))
	return rv
}

func (this *IndexNest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexNest(this)
}

func (this *IndexNest) New() Operator {
	return &IndexNest{}
}

func (this *IndexNest) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *IndexNest) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *IndexNest) Outer() bool {
	return this.outer
}

func (this *IndexNest) For() string {
	return this.keyFor
}

func (this *IndexNest) IdExpr() expression.Expression {
	return this.idExpr
}

func (this *IndexNest) Index() datastore.Index {
	return this.index
}

func (this *IndexNest) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *IndexNest) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "IndexNest"}
	r["namespace"] = this.term.Namespace()
	r["keyspace"] = this.term.Keyspace()
	r["on_key"] = expression.NewStringer().Visit(this.term.Keys())
	r["for"] = this.keyFor

	if this.outer {
		r["outer"] = this.outer
	}

	if this.term.As() != "" {
		r["as"] = this.term.As()
	}

	scan := map[string]interface{}{
		"index":    this.index.Name(),
		"index_id": this.index.Id(),
		"using":    this.index.Type(),
	}

	r["scan"] = scan
	if f != nil {
		f(r)
	}
	return r
}

func (this *IndexNest) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string `json:"#operator"`
		Names string `json:"namespace"`
		Keys  string `json:"keyspace"`
		On    string `json:"on_key"`
		Outer bool   `json:"outer"`
		As    string `json:"as"`
		For   string `json:"for"`
		Scan  struct {
			Index   string              `json:"index"`
			IndexId string              `json:"index_id"`
			Using   datastore.IndexType `json:"using"`
		} `json:"scan"`
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
	this.keyFor = _unmarshalled.For
	this.idExpr = expression.NewField(
		expression.NewMeta(expression.NewIdentifier(this.keyFor)),
		expression.NewFieldName("id", false))
	this.term = algebra.NewKeyspaceTerm(_unmarshalled.Names, _unmarshalled.Keys, _unmarshalled.As, keys_expr, nil)
	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Names, _unmarshalled.Keys)
	if err != nil {
		return err
	}

	this.indexer, err = this.keyspace.Indexer(_unmarshalled.Scan.Using)
	if err != nil {
		return err
	}

	this.index, err = this.indexer.IndexById(_unmarshalled.Scan.IndexId)
	return err
}

func (this *IndexNest) verify(prepared *Prepared) bool {
	return verifyIndex(this.index, this.indexer, prepared)
}
