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

type CreateBucket struct {
	statementBase

	name         string      `json:"name"`
	with         value.Value `json:"with"`
	failIfExists bool        `json:"failIfExists"`
}

func NewCreateBucket(name string, failIfExists bool, with value.Value) *CreateBucket {
	rv := &CreateBucket{
		name:         name,
		with:         with,
		failIfExists: failIfExists,
	}

	rv.stmt = rv
	return rv
}

func (this *CreateBucket) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateBucket(this)
}

func (this *CreateBucket) Signature() value.Value {
	return nil
}

func (this *CreateBucket) Formalize() error {
	return nil
}

func (this *CreateBucket) MapExpressions(mapper expression.Mapper) error {
	return nil
}

func (this *CreateBucket) Expressions() expression.Expressions {
	return nil
}

func (this *CreateBucket) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	privs.Add("", auth.PRIV_CLUSTER_ADMIN, auth.PRIV_PROPS_NONE)

	return privs, nil
}

func (this *CreateBucket) Name() string {
	return this.name
}

func (this *CreateBucket) With() value.Value {
	return this.with
}

func (this *CreateBucket) FailIfExists() bool {
	return this.failIfExists
}

func (this *CreateBucket) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "createBucket"}
	r["name"] = this.name
	if this.with != nil {
		r["with"] = this.with
	}
	r["failIfExists"] = this.failIfExists
	return json.Marshal(r)
}

func (this *CreateBucket) Type() string {
	return "CREATE_BUCKET"
}

func (this *CreateBucket) String() string {
	var s strings.Builder
	s.WriteString("CREATE BUCKET ")

	if !this.failIfExists {
		s.WriteString("IF NOT EXISTS ")
	}

	s.WriteRune('`')
	s.WriteString(this.name)
	s.WriteRune('`')

	if this.with != nil {
		s.WriteString(" WITH ")
		s.WriteString(this.with.String())
	}

	return s.String()
}
