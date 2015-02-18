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

// Create primary index
type CreatePrimaryIndex struct {
	readwrite
	keyspace datastore.Keyspace
	node     *algebra.CreatePrimaryIndex
}

func NewCreatePrimaryIndex(keyspace datastore.Keyspace, node *algebra.CreatePrimaryIndex) *CreatePrimaryIndex {
	return &CreatePrimaryIndex{
		keyspace: keyspace,
		node:     node,
	}
}

func (this *CreatePrimaryIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreatePrimaryIndex(this)
}

func (this *CreatePrimaryIndex) New() Operator {
	return &CreatePrimaryIndex{}
}

func (this *CreatePrimaryIndex) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *CreatePrimaryIndex) Node() *algebra.CreatePrimaryIndex {
	return this.node
}

func (this *CreatePrimaryIndex) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "CreatePrimaryIndex"}
	r["keyspace"] = this.keyspace.Name()
	r["namespace"] = this.keyspace.NamespaceId()
	r["node"] = this.node
	return json.Marshal(r)
}

func (this *CreatePrimaryIndex) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string                      `json:"#operator"`
		Keys  string                      `json:"keyspace"`
		Names string                      `json:"namespace"`
		Node  *algebra.CreatePrimaryIndex `json:"node"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Names, _unmarshalled.Keys)
	return err
}

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
	r := map[string]interface{}{"#operator": "CreateIndex"}
	r["keyspace"] = this.keyspace.Name()
	r["namespace"] = this.keyspace.NamespaceId()
	r["node"] = this.node
	return json.Marshal(r)
}

func (this *CreateIndex) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string               `json:"#operator"`
		Keys  string               `json:"keyspace"`
		Names string               `json:"namespace"`
		Node  *algebra.CreateIndex `json:"node"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Names, _unmarshalled.Keys)
	return err
}

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
	r := map[string]interface{}{"#operator": "DropIndex"}
	r["node"] = this.node
	return json.Marshal(r)
}

func (this *DropIndex) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_    string             `json:"#operator"`
		Node *algebra.DropIndex `json:"node"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	return err
}

// Alter index
type AlterIndex struct {
	readwrite
	index datastore.Index
	node  *algebra.AlterIndex
}

func NewAlterIndex(index datastore.Index, node *algebra.AlterIndex) *AlterIndex {
	return &AlterIndex{
		index: index,
		node:  node,
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

func (this *AlterIndex) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "AlterIndex"}
	r["index"] = this.index.Name()
	r["node"] = this.node
	return json.Marshal(r)
}

func (this *AlterIndex) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string              `json:"#operator"`
		Index string              `json:"index"`
		Node  *algebra.AlterIndex `json:"node"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	return err
	// TODO: recover index from index name
}

// Build indexes
type BuildIndexes struct {
	readwrite
	keyspace datastore.Keyspace
	node     *algebra.BuildIndexes
}

func NewBuildIndexes(keyspace datastore.Keyspace, node *algebra.BuildIndexes) *BuildIndexes {
	return &BuildIndexes{
		keyspace: keyspace,
		node:     node,
	}
}

func (this *BuildIndexes) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitBuildIndexes(this)
}

func (this *BuildIndexes) New() Operator {
	return &BuildIndexes{}
}

func (this *BuildIndexes) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *BuildIndexes) Node() *algebra.BuildIndexes {
	return this.node
}

func (this *BuildIndexes) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "BuildIndexes"}
	r["keyspace"] = this.keyspace.Name()
	r["namespace"] = this.keyspace.NamespaceId()
	r["node"] = this.node
	return json.Marshal(r)
}

func (this *BuildIndexes) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string                `json:"#operator"`
		Keys  string                `json:"keyspace"`
		Names string                `json:"namespace"`
		Node  *algebra.BuildIndexes `json:"node"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Names, _unmarshalled.Keys)
	return err
}
