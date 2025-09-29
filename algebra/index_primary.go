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
Represents the create primary index ddl statement.
Indexes always use case-insensitive matching to
match field names and paths. Index names are unique
per bucket. Type CreatePrimaryIndex is a struct that
contains fields keyspaceref and the using IndexType
string.
*/
type CreatePrimaryIndex struct {
	statementBase

	name         string              `json:"name"`
	keyspace     *KeyspaceRef        `json:"keyspace"`
	partition    *IndexPartitionTerm `json:"partition"`
	using        datastore.IndexType `json:"using"`
	with         value.Value         `json:"with"`
	failIfExists bool                `json:"failIfExists"`
}

/*
The function NewCreatePrimaryIndex returns a pointer
to the CreatePrimaryIndex struct with the input
argument values as fields.
*/
func NewCreatePrimaryIndex(name string, keyspace *KeyspaceRef, partition *IndexPartitionTerm,
	using datastore.IndexType, with value.Value, failIfExists bool) *CreatePrimaryIndex {
	rv := &CreatePrimaryIndex{
		name:         name,
		keyspace:     keyspace,
		partition:    partition,
		using:        using,
		with:         with,
		failIfExists: failIfExists,
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
	f := expression.NewKeyspaceFormalizer(this.keyspace.Keyspace(), nil)
	return this.MapExpressions(f)
}

/*
Returns nil.
*/
func (this *CreatePrimaryIndex) MapExpressions(mapper expression.Mapper) (err error) {
	if this.partition != nil {
		err = this.partition.MapExpressions(mapper)
		if err != nil {
			return
		}
	}
	return
}

/*
Returns all contained Expressions.
*/
func (this *CreatePrimaryIndex) Expressions() expression.Expressions {
	if this.partition != nil && len(this.partition.Expressions()) > 0 {
		return this.partition.Expressions()
	}

	return nil
}

/*
Returns all required privileges.
*/
func (this *CreatePrimaryIndex) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	fullName := this.keyspace.FullName()
	privs.Add(fullName, auth.PRIV_QUERY_CREATE_INDEX, auth.PRIV_PROPS_NONE)

	for _, expr := range this.Expressions() {
		privs.AddAll(expr.Privileges())
	}

	return privs, nil
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
Returns the Partition expression of the create index statement.
*/
func (this *CreatePrimaryIndex) Partition() *IndexPartitionTerm {
	return this.partition
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

func (this *CreatePrimaryIndex) FailIfExists() bool {
	return this.failIfExists
}

/*
Marshals input receiver into byte array.
*/
func (this *CreatePrimaryIndex) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "createPrimaryIndex"}
	r["name"] = this.name
	r["keyspaceRef"] = this.keyspace
	r["using"] = this.using
	if this.partition != nil {
		r["partition"] = this.partition
	}
	if this.with != nil {
		r["with"] = this.with
	}
	r["failIfExists"] = this.failIfExists

	return json.Marshal(r)
}

func (this *CreatePrimaryIndex) Type() string {
	return "CREATE_PRIMARY_INDEX"
}

func (this *CreatePrimaryIndex) String() string {
	var s strings.Builder
	s.WriteString("CREATE PRIMARY INDEX")
	if this.name != "#primary" {
		s.WriteString(" `")
		s.WriteString(this.name)
		s.WriteRune('`')
	}

	if !this.failIfExists {
		s.WriteString(" IF NOT EXISTS")
	}

	s.WriteString(" ON ")
	s.WriteString(this.Keyspace().Path().ProtectedString())

	if this.partition != nil {
		this.partition.writeSyntaxString(&s)
	}

	if this.using != "" && this.using != datastore.DEFAULT {
		s.WriteString(" USING ")
		s.WriteString(strings.ToUpper(string(this.using)))
	}

	if this.with != nil {
		s.WriteString(" WITH ")
		s.WriteString(this.with.String())
	}

	return s.String()
}
