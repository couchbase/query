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

type CreateSequence struct {
	statementBase

	name         *Path       `json:"name"`
	failIfExists bool        `json:"failIfExists"`
	with         value.Value `json:"with"`
}

func NewCreateSequence(name *Path, failIfExists bool, with value.Value) *CreateSequence {
	rv := &CreateSequence{
		name:         name,
		failIfExists: failIfExists,
		with:         with,
	}

	rv.stmt = rv
	return rv
}

func (this *CreateSequence) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateSequence(this)
}

func (this *CreateSequence) Signature() value.Value {
	return nil
}

func (this *CreateSequence) Formalize() error {
	return nil
}

func (this *CreateSequence) MapExpressions(mapper expression.Mapper) error {
	return nil
}

func (this *CreateSequence) Expressions() expression.Expressions {
	return nil
}

func (this *CreateSequence) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	fullName := this.name.ScopePath().FullName()
	privs.Add(fullName, auth.PRIV_QUERY_MANAGE_SEQUENCES, auth.PRIV_PROPS_NONE)
	return privs, nil
}

func (this *CreateSequence) Name() *Path {
	return this.name
}

func (this *CreateSequence) With() value.Value {
	return this.with
}

func (this *CreateSequence) FailIfExists() bool {
	return this.failIfExists
}

func (this *CreateSequence) MarshalName(m map[string]interface{}) {
	m["namespace"] = this.name.Namespace()
	m["bucket"] = this.name.Bucket()
	m["scope"] = this.name.Scope()
	m["keyspace"] = this.name.Keyspace()
}

func (this *CreateSequence) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "createSequence"}
	this.MarshalName(r)
	r["failIfExists"] = this.failIfExists
	if this.with != nil {
		r["with"] = this.with
	}
	return json.Marshal(r)
}

func (this *CreateSequence) Type() string {
	return "CREATE_SEQUENCE"
}

func (this *CreateSequence) String() string {
	var s strings.Builder
	s.WriteString("CREATE SEQUENCE ")
	if !this.failIfExists {
		s.WriteString("IF NOT EXISTS ")
	}
	s.WriteString(this.name.ProtectedString())
	if this.with != nil {
		s.WriteString(" WITH ")
		s.WriteString(this.with.String())
	}
	return s.String()
}
