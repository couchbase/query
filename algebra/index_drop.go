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
	"github.com/couchbaselabs/query/value"
)

type DropIndex struct {
	keyspace *KeyspaceRef `json:"keyspace"`
	name     string       `json:"name"`
}

func NewDropIndex(keyspace *KeyspaceRef, name string) *DropIndex {
	return &DropIndex{
		keyspace: keyspace,
		name:     name,
	}
}

func (this *DropIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropIndex(this)
}

func (this *DropIndex) Signature() value.Value {
	return nil
}

func (this *DropIndex) Formalize() error {
	return nil
}

func (this *DropIndex) Keyspace() *KeyspaceRef {
	return this.keyspace
}

func (this *DropIndex) Name() string {
	return this.name
}
