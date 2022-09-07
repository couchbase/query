//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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

	keyspace     *KeyspaceRef `json:"keyspace"`
	failIfExists bool         `json:"failIfExists"`
	with         value.Value  `json:"with"`
}

/*
The function NewCreateCollection returns a pointer to the
CreateCollection struct with the input argument values as fields.
*/
func NewCreateCollection(keyspace *KeyspaceRef, failIfExists bool, with value.Value) *CreateCollection {
	rv := &CreateCollection{
		keyspace:     keyspace,
		failIfExists: failIfExists,
		with:         with,
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
	fullName := this.keyspace.Path().ScopePath().FullName()
	privs.Add(fullName, auth.PRIV_QUERY_SCOPE_ADMIN, auth.PRIV_PROPS_NONE)

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

func (this *CreateCollection) With() value.Value {
	return this.with
}

/*
Marshals input receiver into byte array.
*/
func (this *CreateCollection) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "createCollection"}
	r["keyspaceRef"] = this.keyspace
	r["failIfExists"] = this.failIfExists
	return json.Marshal(r)
}

func (this *CreateCollection) Type() string {
	return "CREATE_COLLECTION"
}

func (this *CreateCollection) FailIfExists() bool {
	return this.failIfExists
}
