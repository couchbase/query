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

type CreateCatalog struct {
	statementBase

	name         string      `json:"name"`
	catalogType  string      `json:"catalogType"`
	source       string      `json:"source"`
	credential   string      `json:"credential"`
	with         value.Value `json:"with"`
	failIfExists bool        `json:"failIfExists"`
}

func NewCreateCatalog(name, catalogType, source, credential string, failIfExists bool, with value.Value) *CreateCatalog {
	rv := &CreateCatalog{
		name:         name,
		catalogType:  catalogType,
		source:       source,
		credential:   credential,
		with:         with,
		failIfExists: failIfExists,
	}

	rv.stmt = rv
	return rv
}

func (this *CreateCatalog) Accept(visitor Visitor) (any, error) {
	return visitor.VisitCreateCatalog(this)
}

func (this *CreateCatalog) Signature() value.Value {
	return nil
}

func (this *CreateCatalog) Formalize() error {
	return nil
}

func (this *CreateCatalog) MapExpressions(mapper expression.Mapper) error {
	return nil
}

func (this *CreateCatalog) Expressions() expression.Expressions {
	return nil
}

func (this *CreateCatalog) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	privs.Add("", auth.PRIV_CATALOGS_WRITE, auth.PRIV_PROPS_NONE)

	return privs, nil
}

func (this *CreateCatalog) Name() string {
	return this.name
}

func (this *CreateCatalog) CatalogType() string {
	return this.catalogType
}

func (this *CreateCatalog) Source() string {
	return this.source
}

func (this *CreateCatalog) Credential() string {
	return this.credential
}

func (this *CreateCatalog) With() value.Value {
	return this.with
}

func (this *CreateCatalog) FailIfExists() bool {
	return this.failIfExists
}

func (this *CreateCatalog) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "createCatalog"}
	r["name"] = this.name
	r["catalogType"] = this.catalogType
	r["source"] = this.source
	r["credential"] = this.credential
	if this.with != nil {
		r["with"] = this.with
	}
	r["failIfExists"] = this.failIfExists
	return json.Marshal(r)
}

func (this *CreateCatalog) Type() string {
	return "CREATE_CATALOG"
}

func (this *CreateCatalog) String() string {
	var s strings.Builder
	s.WriteString("CREATE CATALOG ")

	if !this.failIfExists {
		s.WriteString("IF NOT EXISTS ")
	}

	s.WriteRune('`')
	s.WriteString(this.name)
	s.WriteRune('`')

	s.WriteString(" TYPE ")
	s.WriteString(this.catalogType)

	if this.source != "" {
		s.WriteString(" SOURCE ")
		s.WriteString(this.source)
	}

	s.WriteString(" AT `")
	s.WriteString(this.credential)
	s.WriteRune('`')

	if this.with != nil {
		s.WriteString(" WITH ")
		s.WriteString(this.with.String())
	}

	return s.String()
}
