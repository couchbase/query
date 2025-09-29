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
	"strings"

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

	keyspace        *KeyspaceRef `json:"keyspace"`
	failIfNotExists bool         `json:"failIfNotExists"`
}

/*
The function NewDropCollection returns a pointer to the
DropCollection struct with the input argument values as fields.
*/
func NewDropCollection(keyspace *KeyspaceRef, failIfNotExists bool) *DropCollection {
	rv := &DropCollection{
		keyspace:        keyspace,
		failIfNotExists: failIfNotExists,
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
	fullName := this.keyspace.Path().ScopePath().FullName()
	privs.Add(fullName, auth.PRIV_QUERY_SCOPE_ADMIN, auth.PRIV_PROPS_NONE)
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
	r["failIfNotExists"] = this.failIfNotExists
	return json.Marshal(r)
}

func (this *DropCollection) Type() string {
	return "DROP_COLLECTION"
}

func (this *DropCollection) FailIfNotExists() bool {
	return this.failIfNotExists
}

func (this *DropCollection) String() string {
	var s strings.Builder
	s.WriteString("DROP COLLECTION ")

	if !this.failIfNotExists {
		s.WriteString("IF EXISTS ")
	}

	s.WriteString(this.keyspace.path.ProtectedString())
	return s.String()
}
