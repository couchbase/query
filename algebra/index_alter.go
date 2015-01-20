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
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type AlterIndex struct {
	statementBase

	keyspace *KeyspaceRef `json:"keyspace"`
	name     string       `json:"name"`
	rename   string       `json:"rename"`
}

func NewAlterIndex(keyspace *KeyspaceRef, name, rename string) *AlterIndex {
	rv := &AlterIndex{
		keyspace: keyspace,
		name:     name,
		rename:   rename,
	}

	rv.stmt = rv
	return rv
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

func (this *AlterIndex) MapExpressions(mapper expression.Mapper) error {
	return nil
}

func (this *AlterIndex) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *AlterIndex) Privileges() (datastore.Privileges, errors.Error) {
	ks, err := datastore.GetKeyspace(this.keyspace.Namespace(), this.keyspace.Keyspace())
	if err != nil {
		return nil, err
	}

	return datastore.Privileges{ks: datastore.PRIV_DDL}, nil
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

func (this *AlterIndex) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "alterIndex"}
	r["keyspaceRef"] = this.keyspace
	r["name"] = this.name
	r["rename"] = this.rename
	return json.Marshal(r)
}
