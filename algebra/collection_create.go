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
Represents the Create collection ddl statement. Type CreateCollection is
a struct that contains fields mapping to each clause in the
create collection statement.
*/
type CreateCollection struct {
	statementBase

	keyspace *KeyspaceRef `json:"keyspace"`
}

/*
The function NewCreateCollection returns a pointer to the
CreateCollection struct with the input argument values as fields.
*/
func NewCreateCollection(keyspace *KeyspaceRef) *CreateCollection {
	rv := &CreateCollection{
		keyspace: keyspace,
	}

	rv.stmt = rv
	return rv
}

/*
It calls the VisitCreateCollection method by passing
in the receiver and returns the interface. It is a
visitor pattern.
*/
func (this *CreateCollection) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateCollection(this)
}

/*
Returns nil.
*/
func (this *CreateCollection) Signature() value.Value {
	return nil
}

/*
Returns nil.
*/
func (this *CreateCollection) Formalize() error {
	return nil
}

/*
This method maps all the constituent clauses.
*/
func (this *CreateCollection) MapExpressions(mapper expression.Mapper) error {
	return nil
}

/*
Return expr from the create collection statement.
*/
func (this *CreateCollection) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *CreateCollection) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	//	fullName := this.keyspace.FullName()
	//	privs.Add(fullName, auth.PRIV_QUERY_CREATE_COLLECTION)

	return privs, nil
}

/*
Returns the keyspace reference of the collection to be created
*/
func (this *CreateCollection) Keyspace() *KeyspaceRef {
	return this.keyspace
}

/*
Returns the name of the collection to be created
*/
func (this *CreateCollection) Name() string {
	return this.keyspace.Path().Keyspace()
}

/*
Marshals input receiver into byte array.
*/
func (this *CreateCollection) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "createCollection"}
	r["keyspaceRef"] = this.keyspace
	return json.Marshal(r)
}

func (this *CreateCollection) Type() string {
	return "CREATE_COLLECTION"
}
