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
	"github.com/couchbaselabs/query/expression/parser"
)

type SendInsert struct {
	readwrite
	keyspace datastore.Keyspace
	key      expression.Expression
	value    expression.Expression
	limit    expression.Expression
}

func NewSendInsert(keyspace datastore.Keyspace, key, value, limit expression.Expression) *SendInsert {
	return &SendInsert{
		keyspace: keyspace,
		key:      key,
		value:    value,
		limit:    limit,
	}
}

func (this *SendInsert) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSendInsert(this)
}

func (this *SendInsert) New() Operator {
	return &SendInsert{}
}

func (this *SendInsert) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *SendInsert) Key() expression.Expression {
	return this.key
}

func (this *SendInsert) Value() expression.Expression {
	return this.value
}

func (this *SendInsert) Limit() expression.Expression {
	return this.limit
}

func (this *SendInsert) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "SendInsert"}
	r["keyspace"] = this.keyspace.Name()
	r["namespace"] = this.keyspace.NamespaceId()

	if this.key != nil {
		r["key"] = this.key.String()
	}

	if this.value != nil {
		r["value"] = this.value.String()
	}

	return json.Marshal(r)
}

func (this *SendInsert) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string `json:"#operator"`
		KeyExpr   string `json:"key"`
		ValueExpr string `json:"value"`
		Keys      string `json:"keyspace"`
		Names     string `json:"namespace"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	if _unmarshalled.KeyExpr != "" {
		this.key, err = parser.Parse(_unmarshalled.KeyExpr)
		if err != nil {
			return err
		}
	}

	if _unmarshalled.ValueExpr != "" {
		this.value, err = parser.Parse(_unmarshalled.ValueExpr)
		if err != nil {
			return err
		}
	}

	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Names, _unmarshalled.Keys)
	return err
}
