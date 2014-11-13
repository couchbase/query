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
	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/expression"
)

type SendInsert struct {
	readwrite
	keyspace datastore.Keyspace
	key      expression.Expression
}

func NewSendInsert(keyspace datastore.Keyspace, key expression.Expression) *SendInsert {
	return &SendInsert{
		keyspace: keyspace,
		key:      key,
	}
}

func (this *SendInsert) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSendInsert(this)
}

func (this *SendInsert) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *SendInsert) Key() expression.Expression {
	return this.key
}

func (this *SendInsert) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "insert"}
	r["keyspace"] = this.keyspace.Name()
	r["key"] = expression.NewStringer().Visit(this.key)
	return json.Marshal(r)
}
