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

type CreateCredentialStore struct {
	statementBase

	name         string      `json:"name"`
	failIfExists bool        `json:"failIfExists"`
	with         value.Value `json:"with"`
}

func NewCreateCredentialStore(name string, failIfExists bool, with value.Value) *CreateCredentialStore {
	rv := &CreateCredentialStore{
		name:         name,
		failIfExists: failIfExists,
		with:         with,
	}

	rv.stmt = rv
	return rv
}

func (this *CreateCredentialStore) Accept(visitor Visitor) (any, error) {
	return visitor.VisitCreateCredentialStore(this)
}

func (this *CreateCredentialStore) Signature() value.Value {
	return nil
}

func (this *CreateCredentialStore) Formalize() error {
	return nil
}

func (this *CreateCredentialStore) MapExpressions(mapper expression.Mapper) error {
	return nil
}

func (this *CreateCredentialStore) Expressions() expression.Expressions {
	return nil
}

func (this *CreateCredentialStore) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	privs.Add("", auth.PRIV_CREDENTIAL_WRITE, auth.PRIV_PROPS_NONE)
	return privs, nil
}

func (this *CreateCredentialStore) Name() string {
	return this.name
}

func (this *CreateCredentialStore) With() value.Value {
	return this.with
}

func (this *CreateCredentialStore) FailIfExists() bool {
	return this.failIfExists
}

func (this *CreateCredentialStore) MarshalJSON() ([]byte, error) {
	r := map[string]any{"type": "createCredentialStore"}
	r["name"] = this.name
	r["failIfExists"] = this.failIfExists
	if this.with != nil {
		r["with"] = this.with
	}
	return json.Marshal(r)
}

func (this *CreateCredentialStore) Type() string {
	return "CREATE_CREDENTIALSTORE"
}

func (this *CreateCredentialStore) String() string {
	var s strings.Builder
	s.WriteString("CREATE CREDENTIALSTORE ")
	if !this.failIfExists {
		s.WriteString("IF NOT EXISTS ")
	}
	s.WriteRune('`')
	s.WriteString(this.name)
	s.WriteRune('`')
	s.WriteString(" WITH ")
	s.WriteString(this.with.String())
	return s.String()
}
