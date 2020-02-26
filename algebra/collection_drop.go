//  Copyright (c) 2020 Couchbase, Inc.
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

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
Represents the Drop collection ddl statement. Type DropCollection is
a struct that contains fields mapping to each clause in the
drop collection statement, namely the keyspace and the collection name.
*/
type DropCollection struct {
	statementBase

	keyspace *KeyspaceRef `json:"keyspace"`
}

/*
The function NewDropCollection returns a pointer to the
DropCollection struct with the input argument values as fields.
*/
func NewDropCollection(keyspace *KeyspaceRef) *DropCollection {
	rv := &DropCollection{
		keyspace: keyspace,
	}

	rv.stmt = rv
	return rv
}

/*
It calls the VisitDropCollection method by passing in the
receiver and returns the interface. It is a visitor
pattern.
*/
func (this *DropCollection) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropCollection(this)
}

/*
Returns nil.
*/
func (this *DropCollection) Signature() value.Value {
	return nil
}

/*
Returns nil.
*/
func (this *DropCollection) Formalize() error {
	return nil
}

/*
Returns nil.
*/
func (this *DropCollection) MapExpressions(mapper expression.Mapper) error {
	return nil
}

/*
Returns all contained Expressions.
*/
func (this *DropCollection) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *DropCollection) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	//	fullName := this.keyspace.FullName()
	//	privs.Add(fullName, auth.PRIV_QUERY_DROP_COLLECTION)
	return privs, nil
}

/*
Return the keyspace reference of the collection to be dropped
*/
func (this *DropCollection) Keyspace() *KeyspaceRef {
	return this.keyspace
}

/*
Returns the name of the collection to be dropped
*/
func (this *DropCollection) Name() string {
	return this.keyspace.Path().Keyspace()
}

/*
Marshals input receiver into byte array.
*/
func (this *DropCollection) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "dropCollection"}
	r["keyspaceRef"] = this.keyspace
	return json.Marshal(r)
}

func (this *DropCollection) Type() string {
	return "DROP_COLLECTION"
}
