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
	"github.com/couchbase/query/value"
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
	return json.Marshal(this.MarshalBase(nil))
}

func (this *AlterIndex) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "AlterIndex"}
	r["index"] = this.index.Name()
	r["index_id"] = this.index.Id()
	r["keyspace"] = this.keyspace.Name()
	r["namespace"] = this.keyspace.NamespaceId()
	r["using"] = this.node.Using()

	if this.node.With() != nil {
		r["with"] = this.node.With()
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *AlterIndex) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_       string              `json:"#operator"`
		Index   string              `json:"index"`
		IndexId string              `json:"index_id"`
		Keys    string              `json:"keyspace"`
		Names   string              `json:"namespace"`
		Using   datastore.IndexType `json:"using"`
		With    json.RawMessage     `json:"with"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	// Build the node
	// Get the keyspace ref (namespace:keyspace)
	ksref := algebra.NewKeyspaceRef(_unmarshalled.Names, _unmarshalled.Keys, "")

	// Get the with clause
	var with value.Value
	if len(_unmarshalled.With) > 0 {
		with = value.NewValue([]byte(_unmarshalled.With))
	}

	this.node = algebra.NewAlterIndex(ksref, _unmarshalled.Index, _unmarshalled.Using, with)

	// Build the index
	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Names, _unmarshalled.Keys)
	if err != nil {
		return err
	}

	// Alter Index is only supported by GSI and doesnt support a USING clause
	indexer, err := this.keyspace.Indexer(_unmarshalled.Using)
	if err != nil {
		return err
	}

	index, err := indexer.IndexById(_unmarshalled.IndexId)
	if err != nil {
		return err
	}

	if _, ok := index.(datastore.AlterIndex); !ok {
		return errors.NewAlterIndexError()
	}

	this.index = index

	return nil
}
