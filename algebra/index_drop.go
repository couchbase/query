//  Copyright 2014-Present Couchbase, Inc.
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

	keyspace        *KeyspaceRef        `json:"keyspace"`
	name            string              `json:"name"`
	using           datastore.IndexType `json:"using"`
	failIfNotExists bool                `json:"failIfNotExists"`
	primaryOnly     bool                `json:"primaryOnly"`
	vector          bool                `json:"vector"`
}

/*
The function NewDropIndex returns a pointer to the
DropIndex struct with the input argument values as fields.
*/
func NewDropIndex(keyspace *KeyspaceRef, name string, using datastore.IndexType, failIfNotExists, primary, vector bool) *DropIndex {
	rv := &DropIndex{
		keyspace:        keyspace,
		name:            name,
		using:           using,
		failIfNotExists: failIfNotExists,
		primaryOnly:     primary,
		vector:          vector,
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
func (this *DropIndex) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	fullName := this.keyspace.FullName()
	if this.using == datastore.FTS {
		privs.Add(fullName, auth.PRIV_SEARCH_DROP_INDEX, auth.PRIV_PROPS_NONE)
	} else {
		privs.Add(fullName, auth.PRIV_QUERY_DROP_INDEX, auth.PRIV_PROPS_NONE)
	}
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

func (this *DropIndex) FailIfNotExists() bool {
	return this.failIfNotExists
}

func (this *DropIndex) PrimaryOnly() bool {
	return this.primaryOnly
}

func (this *DropIndex) Vector() bool {
	return this.vector
}

/*
Marshals input receiver into byte array.
*/
func (this *DropIndex) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "dropIndex"}
	r["keyspaceRef"] = this.keyspace
	r["name"] = this.name
	r["using"] = this.using
	r["failIfNotExists"] = this.failIfNotExists
	r["primaryOnly"] = this.primaryOnly
	return json.Marshal(r)
}

func (this *DropIndex) Type() string {
	return "DROP_INDEX"
}

func (this *DropIndex) String() string {
	var s strings.Builder
	s.WriteString("DROP ")
	if this.vector {
		s.WriteString("VECTOR ")
	} else if this.primaryOnly {
		s.WriteString("PRIMARY ")
	}
	s.WriteString("INDEX")

	if !this.failIfNotExists {
		s.WriteString(" IF EXISTS")
	}

	if !this.primaryOnly || (this.primaryOnly && this.name != "#primary") {
		s.WriteString(" `")
		s.WriteString(this.name)
		s.WriteRune('`')
	}
	s.WriteString(" ON ")
	s.WriteString(this.keyspace.Path().ProtectedString())

	if this.using != "" && this.using != datastore.DEFAULT {
		s.WriteString(" USING ")
		s.WriteString(strings.ToUpper(string(this.using)))
	}

	return s.String()
}
