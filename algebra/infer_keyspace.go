//  Copyright (c) 2016 Couchbase, Inc.
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

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

type InferKeyspace struct {
	statementBase

	keyspace *KeyspaceRef            `json:"keyspace"`
	using    datastore.InferenceType `json:"using"`
	with     value.Value             `json:"with"`
}

func NewInferKeyspace(keyspace *KeyspaceRef, using datastore.InferenceType,
	with value.Value) *InferKeyspace {
	rv := &InferKeyspace{
		keyspace: keyspace,
		using:    using,
		with:     with,
	}

	rv.stmt = rv
	return rv
}

func (this *InferKeyspace) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitInferKeyspace(this)
}

func (this *InferKeyspace) Signature() value.Value {
	return nil
}

func (this *InferKeyspace) Formalize() error {
	return nil
}

func (this *InferKeyspace) MapExpressions(mapper expression.Mapper) error {
	return nil
}

func (this *InferKeyspace) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *InferKeyspace) Privileges() (datastore.Privileges, errors.Error) {
	var privilege datastore.Privilege
	if this.keyspace.Namespace() == "#system" {
		var keyspace = this.keyspace.Keyspace()
		if keyspace == "user_info" {
			privilege = datastore.PRIV_SECURITY_READ
		} else if keyspace == "datastores" || keyspace == "namespace" || keyspace == "dual" {
			// These system tables are open to all.
			return datastore.Privileges{}, nil
		} else {
			privilege = datastore.PRIV_SYSTEM_READ
		}
	} else {
		privilege = datastore.PRIV_READ
	}
	return datastore.Privileges{
		this.keyspace.Namespace() + ":" + this.keyspace.Keyspace(): privilege,
	}, nil
}

func (this *InferKeyspace) Keyspace() *KeyspaceRef {
	return this.keyspace
}

/*
Returns the index type string for the using clause.
*/
func (this *InferKeyspace) Using() datastore.InferenceType {
	return this.using
}

func (this *InferKeyspace) With() value.Value {
	return this.with
}

func (this *InferKeyspace) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "InferKeyspace"}
	r["keyspaceRef"] = this.keyspace
	r["using"] = this.using
	r["with"] = this.with
	return json.Marshal(r)
}
