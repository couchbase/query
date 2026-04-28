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

type DropCatalog struct {
	statementBase

	name            string `json:"name"`
	failIfNotExists bool   `json:"failIfNotExists"`
}

func NewDropCatalog(name string, failIfNotExists bool) *DropCatalog {
	rv := &DropCatalog{
		name:            name,
		failIfNotExists: failIfNotExists,
	}

	rv.stmt = rv
	return rv
}

func (this *DropCatalog) Accept(visitor Visitor) (any, error) {
	return visitor.VisitDropCatalog(this)
}

func (this *DropCatalog) Signature() value.Value {
	return nil
}

func (this *DropCatalog) Formalize() error {
	return nil
}

func (this *DropCatalog) MapExpressions(mapper expression.Mapper) error {
	return nil
}

func (this *DropCatalog) Expressions() expression.Expressions {
	return nil
}

func (this *DropCatalog) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	privs.Add("", auth.PRIV_CATALOGS_WRITE, auth.PRIV_PROPS_NONE)

	return privs, nil
}

func (this *DropCatalog) Name() string {
	return this.name
}

func (this *DropCatalog) FailIfNotExists() bool {
	return this.failIfNotExists
}

func (this *DropCatalog) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "dropCatalog"}
	r["name"] = this.name
	r["failIfNotExists"] = this.failIfNotExists
	return json.Marshal(r)
}

func (this *DropCatalog) Type() string {
	return "DROP_CATALOG"
}

func (this *DropCatalog) String() string {
	var s strings.Builder
	s.WriteString("DROP CATALOG ")

	if !this.failIfNotExists {
		s.WriteString("IF EXISTS ")
	}

	s.WriteRune('`')
	s.WriteString(this.name)
	s.WriteRune('`')

	return s.String()
}
