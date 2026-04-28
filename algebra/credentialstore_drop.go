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

type DropCredentialStore struct {
	statementBase

	name            string `json:"name"`
	failIfNotExists bool   `json:"failIfNotExists"`
}

func NewDropCredentialStore(name string, failIfNotExists bool) *DropCredentialStore {
	rv := &DropCredentialStore{
		name:            name,
		failIfNotExists: failIfNotExists,
	}

	rv.stmt = rv
	return rv
}

func (this *DropCredentialStore) Accept(visitor Visitor) (any, error) {
	return visitor.VisitDropCredentialStore(this)
}

func (this *DropCredentialStore) Signature() value.Value {
	return nil
}

func (this *DropCredentialStore) Formalize() error {
	return nil
}

func (this *DropCredentialStore) MapExpressions(mapper expression.Mapper) error {
	return nil
}

func (this *DropCredentialStore) Expressions() expression.Expressions {
	return nil
}

func (this *DropCredentialStore) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	privs.Add("", auth.PRIV_SECURITY_WRITE, auth.PRIV_PROPS_NONE)
	return privs, nil
}

func (this *DropCredentialStore) Name() string {
	return this.name
}

func (this *DropCredentialStore) FailIfNotExists() bool {
	return this.failIfNotExists
}

func (this *DropCredentialStore) MarshalJSON() ([]byte, error) {
	r := map[string]any{"type": "dropCredentialStore"}
	r["name"] = this.name
	r["failIfNotExists"] = this.failIfNotExists
	return json.Marshal(r)
}

func (this *DropCredentialStore) Type() string {
	return "DROP_CREDENTIALSTORE"
}

func (this *DropCredentialStore) String() string {
	var s strings.Builder
	s.WriteString("DROP CREDENTIALSTORE ")
	if !this.failIfNotExists {
		s.WriteString("IF EXISTS ")
	}
	s.WriteRune('`')
	s.WriteString(this.name)
	s.WriteRune('`')
	return s.String()
}
