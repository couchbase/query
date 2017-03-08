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
Represents the Drop index ddl statement. Type DropIndex is
a struct that contains fields mapping to each clause in the
drop index statement, namely the keyspace and the index name.
*/
type DropIndex struct {
	statementBase

	keyspace *KeyspaceRef        `json:"keyspace"`
	name     string              `json:"name"`
	using    datastore.IndexType `json:"using"`
}

/*
The function NewDropIndex returns a pointer to the
DropIndex struct with the input argument values as fields.
*/
func NewDropIndex(keyspace *KeyspaceRef, name string, using datastore.IndexType) *DropIndex {
	rv := &DropIndex{
		keyspace: keyspace,
		name:     name,
		using:    using,
	}

	rv.stmt = rv
	return rv
}

/*
It calls the VisitDropIndex method by passing in the
receiver and returns the interface. It is a visitor
pattern.
*/
func (this *DropIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropIndex(this)
}

/*
Returns nil.
*/
func (this *DropIndex) Signature() value.Value {
	return nil
}

/*
Returns nil.
*/
func (this *DropIndex) Formalize() error {
	return nil
}

/*
Returns nil.
*/
func (this *DropIndex) MapExpressions(mapper expression.Mapper) error {
	return nil
}

/*
Returns all contained Expressions.
*/
func (this *DropIndex) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *DropIndex) Privileges() (*datastore.Privileges, errors.Error) {
	privs := datastore.NewPrivileges()
	fullName := this.keyspace.FullName()
	privs.Add(fullName, datastore.PRIV_QUERY_DROP_INDEX)
	return privs, nil
}

/*
Return the keyspace.
*/
func (this *DropIndex) Keyspace() *KeyspaceRef {
	return this.keyspace
}

/*
Return the name of the index to be dropped.
*/
func (this *DropIndex) Name() string {
	return this.name
}

/*
Returns the index type string for the using clause.
*/
func (this *DropIndex) Using() datastore.IndexType {
	return this.using
}

/*
Marshals input receiver into byte array.
*/
func (this *DropIndex) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "dropIndex"}
	r["keyspaceRef"] = this.keyspace
	r["name"] = this.name
	r["using"] = this.using
	return json.Marshal(r)
}
