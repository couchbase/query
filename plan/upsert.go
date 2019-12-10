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

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type SendUpsert struct {
	readwrite
	keyspace datastore.Keyspace
	alias    string
	key      expression.Expression
	value    expression.Expression
}

func NewSendUpsert(keyspace datastore.Keyspace, alias string, key, value expression.Expression) *SendUpsert {
	return &SendUpsert{
		keyspace: keyspace,
		alias:    alias,
		key:      key,
		value:    value,
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

func (this *SendUpsert) Alias() string {
	return this.alias
}

func (this *SendUpsert) Key() expression.Expression {
	return this.key
}

func (this *SendUpsert) Value() expression.Expression {
	return this.value
}

func (this *SendUpsert) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *SendUpsert) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "SendUpsert"}
	r["keyspace"] = this.keyspace.Name()
	r["namespace"] = this.keyspace.NamespaceId()
	r["alias"] = this.alias

	if this.key != nil {
		r["key"] = this.key.String()
	}

	if this.value != nil {
		r["value"] = this.value.String()
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *SendUpsert) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string `json:"#operator"`
		KeyExpr   string `json:"key"`
		ValueExpr string `json:"value"`
		Keys      string `json:"keyspace"`
		Names     string `json:"namespace"`
		Alias     string `json:"alias"`
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

	this.alias = _unmarshalled.Alias
	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Names, _unmarshalled.Keys)
	return err
}

func (this *SendUpsert) verify(prepared *Prepared) bool {
	var res bool

	this.keyspace, res = verifyKeyspace(this.keyspace, prepared)
	return res
}
