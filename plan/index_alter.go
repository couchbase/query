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

// Alter index
type AlterIndex struct {
	readwrite
	index    datastore.Index
	node     *algebra.AlterIndex
	keyspace datastore.Keyspace
}

func NewAlterIndex(index datastore.Index, node *algebra.AlterIndex,
	keyspace datastore.Keyspace) *AlterIndex {
	return &AlterIndex{
		index:    index,
		node:     node,
		keyspace: keyspace,
	}
}

func (this *AlterIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAlterIndex(this)
}

func (this *AlterIndex) New() Operator {
	return &AlterIndex{}
}

func (this *AlterIndex) Index() datastore.Index {
	return this.index
}

func (this *AlterIndex) Node() *algebra.AlterIndex {
	return this.node
}

func (this *AlterIndex) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *AlterIndex) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "AlterIndex"}
	r["index"] = this.index.Name()
	r["index_id"] = this.index.Id()
	r["keyspace"] = this.keyspace.Name()
	r["namespace"] = this.keyspace.NamespaceId()
	r["rename"] = this.node.Rename()
	r["using"] = this.node.Using()
	return json.Marshal(r)
}

func (this *AlterIndex) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_       string              `json:"#operator"`
		Index   string              `json:"index"`
		IndexId string              `json:"index_id"`
		Keys    string              `json:"keyspace"`
		Names   string              `json:"namespace"`
		Rename  string              `json:"rename"`
		Using   datastore.IndexType `json:"using"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	ksref := algebra.NewKeyspaceRef(_unmarshalled.Names, _unmarshalled.Keys, "")
	this.node = algebra.NewAlterIndex(ksref, _unmarshalled.Index, _unmarshalled.Using, _unmarshalled.Rename)

	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Names, _unmarshalled.Keys)
	if err != nil {
		return err
	}

	indexer, err := this.keyspace.Indexer(_unmarshalled.Using)
	if err != nil {
		return err
	}

	this.index, err = indexer.IndexById(_unmarshalled.IndexId)
	return err
}
