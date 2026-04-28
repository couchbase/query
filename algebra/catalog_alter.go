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

type AlterCatalog struct {
	statementBase

	name string      `json:"name"`
	with value.Value `json:"with"`
}

func NewAlterCatalog(name string, with value.Value) *AlterCatalog {
	rv := &AlterCatalog{
		name: name,
		with: with,
	}

	rv.stmt = rv
	return rv
}

func (this *AlterCatalog) Accept(visitor Visitor) (any, error) {
	return visitor.VisitAlterCatalog(this)
}

func (this *AlterCatalog) Signature() value.Value {
	return nil
}

func (this *AlterCatalog) Formalize() error {
	return nil
}

func (this *AlterCatalog) MapExpressions(mapper expression.Mapper) error {
	return nil
}

func (this *AlterCatalog) Expressions() expression.Expressions {
	return nil
}

func (this *AlterCatalog) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	privs.Add("", auth.PRIV_CATALOGS_WRITE, auth.PRIV_PROPS_NONE)

	return privs, nil
}

func (this *AlterCatalog) Name() string {
	return this.name
}

func (this *AlterCatalog) With() value.Value {
	return this.with
}

func (this *AlterCatalog) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "alterCatalog"}
	r["name"] = this.name
	if this.with != nil {
		r["with"] = this.with
	}
	return json.Marshal(r)
}

func (this *AlterCatalog) Type() string {
	return "ALTER_CATALOG"
}

func (this *AlterCatalog) String() string {
	var s strings.Builder
	s.WriteString("ALTER CATALOG `")
	s.WriteString(this.name)
	s.WriteRune('`')

	if this.with != nil {
		s.WriteString(" WITH ")
		s.WriteString(this.with.String())
	}

	return s.String()
}
