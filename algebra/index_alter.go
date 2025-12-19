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

type AlterIndex struct {
	statementBase

	keyspace *KeyspaceRef        `json:"keyspace"`
	name     string              `json:"name"`
	using    datastore.IndexType `json:"using"`
	with     value.Value         `json:"with"`
}

func NewAlterIndex(keyspace *KeyspaceRef, name string, using datastore.IndexType, with value.Value) *AlterIndex {
	rv := &AlterIndex{
		keyspace: keyspace,
		name:     name,
		using:    using,
		with:     with,
	}

	rv.stmt = rv
	return rv
}

func (this *AlterIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAlterIndex(this)
}

func (this *AlterIndex) Signature() value.Value {
	return nil
}

func (this *AlterIndex) Formalize() error {
	return nil
}

func (this *AlterIndex) MapExpressions(mapper expression.Mapper) error {
	return nil
}

func (this *AlterIndex) Expressions() expression.Expressions {
	return nil
}

/*
Returns all required privileges.
*/
func (this *AlterIndex) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	fullName := this.keyspace.FullName()
	privs.Add(fullName, auth.PRIV_QUERY_ALTER_INDEX, auth.PRIV_PROPS_NONE)
	return privs, nil
}

func (this *AlterIndex) Keyspace() *KeyspaceRef {
	return this.keyspace
}

func (this *AlterIndex) Name() string {
	return this.name
}

func (this *AlterIndex) Using() datastore.IndexType {
	return this.using
}

func (this *AlterIndex) With() value.Value {
	return this.with
}

func (this *AlterIndex) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "alterIndex"}
	r["keyspaceRef"] = this.keyspace
	r["name"] = this.name
	r["using"] = this.using
	if this.with != nil {
		r["with"] = this.with
	}
	return json.Marshal(r)
}

func (this *AlterIndex) Type() string {
	return "ALTER_INDEX"
}

func (this *AlterIndex) String() string {
	var s strings.Builder
	s.WriteString("ALTER INDEX `")
	s.WriteString(this.name)
	s.WriteString("` ON ")
	s.WriteString(this.keyspace.Path().ProtectedString())

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
