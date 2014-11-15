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
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type CreateIndex struct {
	name      string                 `json:"name"`
	keyspace  *KeyspaceRef           `json:"keyspace"`
	exprs     expression.Expressions `json:"expressions"`
	partition expression.Expression  `json:"partition"`
	where     expression.Expression  `json:"where"`
	using     datastore.IndexType    `json:"using"`
}

func NewCreateIndex(name string, keyspace *KeyspaceRef, exprs expression.Expressions,
	partition, where expression.Expression, using datastore.IndexType) *CreateIndex {
	return &CreateIndex{
		name:      name,
		keyspace:  keyspace,
		exprs:     exprs,
		partition: partition,
		where:     where,
		using:     using,
	}
}

func (this *CreateIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateIndex(this)
}

func (this *CreateIndex) Signature() value.Value {
	return nil
}

func (this *CreateIndex) Formalize() error {
	return nil
}

func (this *CreateIndex) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.exprs.MapExpressions(mapper)
	if err != nil {
		return
	}

	if this.partition != nil {
		this.partition, err = mapper.Map(this.partition)
		if err != nil {
			return
		}
	}

	if this.where != nil {
		this.where, err = mapper.Map(this.where)
		if err != nil {
			return
		}
	}

	return
}

func (this *CreateIndex) Name() string {
	return this.name
}

func (this *CreateIndex) Keyspace() *KeyspaceRef {
	return this.keyspace
}

func (this *CreateIndex) Expressions() expression.Expressions {
	return this.exprs
}

func (this *CreateIndex) Partition() expression.Expression {
	return this.partition
}

func (this *CreateIndex) Where() expression.Expression {
	return this.where
}

func (this *CreateIndex) Using() datastore.IndexType {
	return this.using
}

func (this *CreateIndex) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "createIndex"}
	r["keyspaceRef"] = this.keyspace
	r["name"] = this.name
	if this.partition != nil {
		r["partition"] = expression.NewStringer().Visit(this.partition)
	}
	r["using"] = this.using
	if this.where != nil {
		r["where"] = expression.NewStringer().Visit(this.where)
	}
	return json.Marshal(r)
}
