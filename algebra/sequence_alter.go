//  Copyright 2023-Present Couchbase, Inc.
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

type AlterSequence struct {
	statementBase

	name *Path       `json:"name"`
	with value.Value `json:"with"`
}

func NewAlterSequence(name *Path, with value.Value) *AlterSequence {
	rv := &AlterSequence{
		name: name,
		with: with,
	}

	rv.stmt = rv
	return rv
}

func (this *AlterSequence) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAlterSequence(this)
}

func (this *AlterSequence) Signature() value.Value {
	return nil
}

func (this *AlterSequence) Formalize() error {
	return nil
}

func (this *AlterSequence) MapExpressions(mapper expression.Mapper) error {
	return nil
}

func (this *AlterSequence) Expressions() expression.Expressions {
	return nil
}

func (this *AlterSequence) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	fullName := this.name.ScopePath().FullName()
	privs.Add(fullName, auth.PRIV_QUERY_MANAGE_SEQUENCES, auth.PRIV_PROPS_NONE)
	return privs, nil
}

func (this *AlterSequence) Name() *Path {
	return this.name
}

func (this *AlterSequence) With() value.Value {
	return this.with
}

func (this *AlterSequence) MarshalName(m map[string]interface{}) {
	m["namespace"] = this.name.Namespace()
	m["bucket"] = this.name.Bucket()
	m["scope"] = this.name.Scope()
	m["keyspace"] = this.name.Keyspace()
}

func (this *AlterSequence) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "alterSequence"}
	this.MarshalName(r)
	if this.with != nil {
		r["with"] = this.with
	}
	return json.Marshal(r)
}

func (this *AlterSequence) Type() string {
	return "ALTER_SEQUENCE"
}

func (this *AlterSequence) String() string {
	var s strings.Builder
	s.WriteString("ALTER SEQUENCE ")
	s.WriteString(this.name.ProtectedString())
	if this.with != nil {
		s.WriteString(" WITH ")
		s.WriteString(this.with.String())
	}
	return s.String()
}
