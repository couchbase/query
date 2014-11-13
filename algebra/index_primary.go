//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"encoding/json"
	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/value"
)

type CreatePrimaryIndex struct {
	keyspace *KeyspaceRef        `json:"keyspace"`
	using    datastore.IndexType `json:"using"`
}

func NewCreatePrimaryIndex(keyspace *KeyspaceRef, using datastore.IndexType) *CreatePrimaryIndex {
	return &CreatePrimaryIndex{
		keyspace: keyspace,
		using:    using,
	}
}

func (this *CreatePrimaryIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreatePrimaryIndex(this)
}

func (this *CreatePrimaryIndex) Signature() value.Value {
	return nil
}

func (this *CreatePrimaryIndex) Formalize() error {
	return nil
}

func (this *CreatePrimaryIndex) Keyspace() *KeyspaceRef {
	return this.keyspace
}

func (this *CreatePrimaryIndex) Using() datastore.IndexType {
	return this.using
}

func (this *CreatePrimaryIndex) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "createPrimaryIndex"}
	r["keyspaceRef"] = this.keyspace
	r["using"] = this.using
	return json.Marshal(r)
}
