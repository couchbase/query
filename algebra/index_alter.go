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

type AlterIndex struct {
	keyspace *KeyspaceRef `json:"keyspace"`
	name     string       `json:"name"`
	rename   string       `json:"rename"`
}

func NewAlterIndex(keyspace *KeyspaceRef, name, rename string) *AlterIndex {
	return &AlterIndex{
		keyspace: keyspace,
		name:     name,
		rename:   rename,
	}
}

func (this *AlterIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAlterIndex(this)
}

func (this *AlterIndex) Signature() value.Value {
	return nil
}

func (this *AlterIndex) Formalize() error {
	return nil
}

func (this *AlterIndex) Keyspace() *KeyspaceRef {
	return this.keyspace
}

func (this *AlterIndex) Name() string {
	return this.name
}

func (this *AlterIndex) Rename() string {
	return this.rename
}
