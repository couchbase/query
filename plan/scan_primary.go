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

type PrimaryScan struct {
	readonly
	index    datastore.PrimaryIndex
	indexer  datastore.Indexer
	keyspace datastore.Keyspace
	term     *algebra.KeyspaceTerm
	limit    expression.Expression
}

func NewPrimaryScan(index datastore.PrimaryIndex, keyspace datastore.Keyspace,
	term *algebra.KeyspaceTerm, limit expression.Expression) *PrimaryScan {
	return &PrimaryScan{
		index:    index,
		indexer:  getIndexer(term.Namespace(), term.Keyspace(), index.Type()),
		keyspace: keyspace,
		term:     term,
		limit:    limit,
	}
}

func (this *PrimaryScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitPrimaryScan(this)
}

func (this *PrimaryScan) New() Operator {
	return &PrimaryScan{}
}

func (this *PrimaryScan) Index() datastore.PrimaryIndex {
	return this.index
}

func (this *PrimaryScan) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *PrimaryScan) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *PrimaryScan) Limit() expression.Expression {
	return this.limit
}

func (this *PrimaryScan) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *PrimaryScan) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "PrimaryScan"}
	r["index"] = this.index.Name()
	r["namespace"] = this.term.Namespace()
	r["keyspace"] = this.term.Keyspace()
	r["using"] = this.index.Type()

	if this.term.As() != "" {
		r["as"] = this.term.As()
	}

	if this.limit != nil {
		r["limit"] = expression.NewStringer().Visit(this.limit)
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *PrimaryScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string              `json:"#operator"`
		Index string              `json:"index"`
		Names string              `json:"namespace"`
		Keys  string              `json:"keyspace"`
		As    string              `json:"as"`
		Using datastore.IndexType `json:"using"`
		Limit string              `json:"limit"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	if _unmarshalled.Limit != "" {
		this.limit, err = parser.Parse(_unmarshalled.Limit)
		if err != nil {
			return err
		}
	}

	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Names, _unmarshalled.Keys)
	if err != nil {
		return err
	}

	this.term = algebra.NewKeyspaceTerm(_unmarshalled.Names, _unmarshalled.Keys, _unmarshalled.As, nil, nil)
	this.indexer, err = this.keyspace.Indexer(_unmarshalled.Using)
	if err != nil {
		return err
	}

	index, err := this.indexer.IndexByName(_unmarshalled.Index)
	if err != nil {
		return err
	}

	primary, ok := index.(datastore.PrimaryIndex)
	if ok {
		this.index = primary
		return nil
	}

	return fmt.Errorf("Unable to unmarshal %s as primary index.", _unmarshalled.Index)
}

func (this *PrimaryScan) verify(prepared *Prepared) bool {
	return verifyIndex(this.index, this.indexer, prepared)
}
