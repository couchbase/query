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

// Drop index
type DropIndex struct {
	readwrite
	index datastore.Index
	node  *algebra.DropIndex
}

func NewDropIndex(index datastore.Index, node *algebra.DropIndex) *DropIndex {
	return &DropIndex{
		index: index,
		node:  node,
	}
}

func (this *DropIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropIndex(this)
}

func (this *DropIndex) New() Operator {
	return &DropIndex{}
}

func (this *DropIndex) Index() datastore.Index {
	return this.index
}

func (this *DropIndex) Node() *algebra.DropIndex {
	return this.node
}

func (this *DropIndex) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *DropIndex) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "DropIndex"}
	r["namespace"] = this.node.Keyspace().Namespace()
	r["keyspace"] = this.node.Keyspace().Keyspace()
	r["using"] = this.node.Using()
	r["name"] = this.node.Name()
	r["index_id"] = this.index.Id()
	if f != nil {
		f(r)
	}
	return r
}

func (this *DropIndex) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string              `json:"#operator"`
		Namespace string              `json:"namespace"`
		Keyspace  string              `json:"keyspace"`
		Using     datastore.IndexType `json:"using"`
		Name      string              `json:"name"`
		IndexId   string              `json:"index_id"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	// Build this.node.
	ksref := algebra.NewKeyspaceRef(_unmarshalled.Namespace, _unmarshalled.Keyspace, "") // Parameters: namespace, keyspace, as.
	this.node = algebra.NewDropIndex(ksref, _unmarshalled.Name, _unmarshalled.Using)

	// Build this.index.
	keyspace, err := datastore.GetKeyspace(_unmarshalled.Namespace, _unmarshalled.Keyspace)
	if err != nil {
		return err
	}
	indexer, err := keyspace.Indexer(_unmarshalled.Using)
	if err != nil {
		return err
	}
	index, err := indexer.IndexById(_unmarshalled.IndexId)
	if err != nil {
		return err
	}
	this.index = index

	return nil
}
