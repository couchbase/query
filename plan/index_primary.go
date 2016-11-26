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
	return json.Marshal(this.MarshalBase(nil))
}

func (this *CreatePrimaryIndex) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "CreatePrimaryIndex"}
	r["keyspace"] = this.keyspace.Name()
	r["namespace"] = this.keyspace.NamespaceId()
	r["node"] = this.node
	if f != nil {
		f(r)
	}
	return r
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
