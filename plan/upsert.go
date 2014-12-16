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

type SendUpsert struct {
	readwrite
	keyspace datastore.Keyspace
	key      expression.Expression
}

func NewSendUpsert(keyspace datastore.Keyspace, key expression.Expression) *SendUpsert {
	return &SendUpsert{
		keyspace: keyspace,
		key:      key,
	}
}

func (this *SendUpsert) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSendUpsert(this)
}

func (this *SendUpsert) New() Operator {
	return &SendUpsert{}
}

func (this *SendUpsert) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *SendUpsert) Key() expression.Expression {
	return this.key
}

func (this *SendUpsert) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "SendUpsert"}
	r["key"] = expression.NewStringer().Visit(this.key)
	r["keyspace"] = this.keyspace.Name()
	r["namespace"] = this.keyspace.NamespaceId()
	return json.Marshal(r)
}

func (this *SendUpsert) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_       string `json:"#operator"`
		KeyExpr string `json:"key"`
		Keys    string `json:"keyspace"`
		Names   string `json:"namespace"`
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

	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Names, _unmarshalled.Keys)
	return nil
}
