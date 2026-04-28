//  Copyright 2026-Present Couchbase, Inc.
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
Represents the Alter collection ddl statement.
ALTER COLLECTION keyspace WITH { ... }
*/
type AlterCollection struct {
	statementBase

	keyspace *KeyspaceRef `json:"keyspace"`
	with     value.Value  `json:"with"`
}

/*
NewAlterCollection returns a pointer to the AlterCollection struct.
*/
func NewAlterCollection(keyspace *KeyspaceRef, with value.Value) *AlterCollection {
	rv := &AlterCollection{
		keyspace: keyspace,
		with:     with,
	}

	rv.stmt = rv
	return rv
}

func (this *AlterCollection) Accept(visitor Visitor) (any, error) {
	return visitor.VisitAlterCollection(this)
}

func (this *AlterCollection) Signature() value.Value {
	return nil
}

func (this *AlterCollection) Formalize() error {
	return nil
}

func (this *AlterCollection) MapExpressions(mapper expression.Mapper) error {
	return nil
}

func (this *AlterCollection) Expressions() expression.Expressions {
	return nil
}

func (this *AlterCollection) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	fullName := this.keyspace.Path().ScopePath().FullName()
	privs.Add(fullName, auth.PRIV_QUERY_SCOPE_ADMIN, auth.PRIV_PROPS_NONE)
	return privs, nil
}

func (this *AlterCollection) Keyspace() *KeyspaceRef {
	return this.keyspace
}

func (this *AlterCollection) Name() string {
	return this.keyspace.Path().Keyspace()
}

func (this *AlterCollection) With() value.Value {
	return this.with
}

func (this *AlterCollection) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "alterCollection"}
	r["keyspaceRef"] = this.keyspace
	if this.with != nil {
		r["with"] = this.with
	}
	return json.Marshal(r)
}

func (this *AlterCollection) Type() string {
	return "ALTER_COLLECTION"
}

func (this *AlterCollection) String() string {
	var s strings.Builder
	s.WriteString("ALTER COLLECTION ")
	s.WriteString(this.keyspace.Path().ProtectedString())

	if this.with != nil {
		s.WriteString(" WITH ")
		s.WriteString(this.with.String())
	}

	return s.String()
}
