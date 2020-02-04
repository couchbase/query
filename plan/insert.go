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

type SendInsert struct {
	readwrite
	keyspace datastore.Keyspace
	alias    string
	key      expression.Expression
	value    expression.Expression
	options  expression.Expression
	limit    expression.Expression
}

func NewSendInsert(keyspace datastore.Keyspace, alias string,
	key, value, options, limit expression.Expression) *SendInsert {
	return &SendInsert{
		keyspace: keyspace,
		alias:    alias,
		key:      key,
		value:    value,
		options:  options,
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

func (this *SendInsert) Alias() string {
	return this.alias
}

func (this *SendInsert) Key() expression.Expression {
	return this.key
}

func (this *SendInsert) Value() expression.Expression {
	return this.value
}

func (this *SendInsert) Options() expression.Expression {
	return this.options
}

func (this *SendInsert) Limit() expression.Expression {
	return this.limit
}

func (this *SendInsert) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *SendInsert) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "SendInsert"}
	r["keyspace"] = this.keyspace.Name()
	r["namespace"] = this.keyspace.NamespaceId()
	r["alias"] = this.alias

	if this.limit != nil {
		r["limit"] = this.limit
	}

	if this.key != nil {
		r["key"] = this.key.String()
	}

	if this.value != nil {
		r["value"] = this.value.String()
	}

	if this.options != nil {
		r["options"] = this.options.String()
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *SendInsert) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string `json:"#operator"`
		KeyExpr     string `json:"key"`
		ValueExpr   string `json:"value"`
		OptionsExpr string `json:"options"`
		Keys        string `json:"keyspace"`
		Names       string `json:"namespace"`
		Alias       string `json:"alias"`
		Limit       string `json:"limit"`
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

	if _unmarshalled.OptionsExpr != "" {
		this.options, err = parser.Parse(_unmarshalled.OptionsExpr)
		if err != nil {
			return err
		}
	}

	this.alias = _unmarshalled.Alias

	if _unmarshalled.Limit != "" {
		this.limit, err = parser.Parse(_unmarshalled.Limit)
		if err != nil {
			return err
		}
	}

	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Names, _unmarshalled.Keys)
	return err
}

func (this *SendInsert) verify(prepared *Prepared) bool {
	var res bool

	this.keyspace, res = verifyKeyspace(this.keyspace, prepared)
	return res
}
