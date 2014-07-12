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
	"github.com/couchbaselabs/query/catalog"
	"github.com/couchbaselabs/query/expression"
)

type CreateIndex struct {
	name      string                 `json:"name"`
	keyspace  *KeyspaceRef           `json:"keyspace"`
	exprs     expression.Expressions `json:"expressions"`
	partition expression.Expression  `json:"partition"`
	using     catalog.IndexType      `json:"using"`
}

func NewCreateIndex(name string, keyspace *KeyspaceRef, exprs expression.Expressions,
	partition expression.Expression, using catalog.IndexType) *CreateIndex {
	return &CreateIndex{
		name:      name,
		keyspace:  keyspace,
		exprs:     exprs,
		partition: partition,
		using:     using,
	}
}

func (this *CreateIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateIndex(this)
}
