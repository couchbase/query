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

type AlterCredentialStore struct {
	statementBase

	name string      `json:"name"`
	with value.Value `json:"with"`
}

func NewAlterCredentialStore(name string, with value.Value) *AlterCredentialStore {
	rv := &AlterCredentialStore{
		name: name,
		with: with,
	}

	rv.stmt = rv
	return rv
}

func (this *AlterCredentialStore) Accept(visitor Visitor) (any, error) {
	return visitor.VisitAlterCredentialStore(this)
}

func (this *AlterCredentialStore) Signature() value.Value {
	return nil
}

func (this *AlterCredentialStore) Formalize() error {
	return nil
}

func (this *AlterCredentialStore) MapExpressions(mapper expression.Mapper) error {
	return nil
}

func (this *AlterCredentialStore) Expressions() expression.Expressions {
	return nil
}

func (this *AlterCredentialStore) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	privs.Add("", auth.PRIV_CREDENTIAL_WRITE, auth.PRIV_PROPS_NONE)
	return privs, nil
}

func (this *AlterCredentialStore) Name() string {
	return this.name
}

func (this *AlterCredentialStore) With() value.Value {
	return this.with
}

func (this *AlterCredentialStore) MarshalJSON() ([]byte, error) {
	r := map[string]any{"type": "alterCredentialStore"}
	r["name"] = this.name
	if this.with != nil {
		r["with"] = this.with
	}
	return json.Marshal(r)
}

func (this *AlterCredentialStore) Type() string {
	return "ALTER_CREDENTIALSTORE"
}

func (this *AlterCredentialStore) String() string {
	var s strings.Builder
	s.WriteString("ALTER CREDENTIALSTORE `")
	s.WriteString(this.name)
	s.WriteRune('`')
	if this.with != nil {
		s.WriteString(" WITH ")
		s.WriteString(this.with.String())
	}
	return s.String()
}
