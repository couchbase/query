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

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
Represents the create primary index ddl statement.
Indexes always use case-insensitive matching to
match field names and paths. Index names are unique
per bucket. Type CreatePrimaryIndex is a struct that
contains fields keyspaceref and the using IndexType
string.
*/
type CreatePrimaryIndex struct {
	statementBase

	name     string              `json:"name"`
	keyspace *KeyspaceRef        `json:"keyspace"`
	using    datastore.IndexType `json:"using"`
	with     value.Value         `json:"with"`
}

/*
The function NewCreatePrimaryIndex returns a pointer
to the CreatePrimaryIndex struct with the input
argument values as fields.
*/
func NewCreatePrimaryIndex(name string, keyspace *KeyspaceRef,
	using datastore.IndexType, with value.Value) *CreatePrimaryIndex {
	rv := &CreatePrimaryIndex{
		name:     name,
		keyspace: keyspace,
		using:    using,
		with:     with,
	}

	rv.stmt = rv
	return rv
}

/*
It calls the VisitCreatePrimaryIndex method by passing
in the receiver and returns the interface. It is a
visitor pattern.
*/
func (this *CreatePrimaryIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreatePrimaryIndex(this)
}

/*
Returns nil.
*/
func (this *CreatePrimaryIndex) Signature() value.Value {
	return nil
}

/*
Returns nil.
*/
func (this *CreatePrimaryIndex) Formalize() error {
	return nil
}

/*
Returns nil.
*/
func (this *CreatePrimaryIndex) MapExpressions(mapper expression.Mapper) error {
	return nil
}

/*
Returns all contained Expressions.
*/
func (this *CreatePrimaryIndex) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *CreatePrimaryIndex) Privileges() (datastore.Privileges, errors.Error) {
	return datastore.Privileges{
		this.keyspace.Namespace() + ":" + this.keyspace.Keyspace(): datastore.PRIV_DDL,
	}, nil
}

/*
Index name.
*/
func (this *CreatePrimaryIndex) Name() string {
	return this.name
}

/*
Returns the input keyspace.
*/
func (this *CreatePrimaryIndex) Keyspace() *KeyspaceRef {
	return this.keyspace
}

/*
Returns the index type string for the using clause.
*/
func (this *CreatePrimaryIndex) Using() datastore.IndexType {
	return this.using
}

/*
Returns the WITH deployment plan.
*/
func (this *CreatePrimaryIndex) With() value.Value {
	return this.with
}

/*
Marshals input receiver into byte array.
*/
func (this *CreatePrimaryIndex) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "createPrimaryIndex"}
	r["name"] = this.name
	r["keyspaceRef"] = this.keyspace
	r["using"] = this.using
	if this.with != nil {
		r["with"] = this.with
	}

	return json.Marshal(r)
}
