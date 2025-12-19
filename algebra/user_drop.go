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

type DropUser struct {
	statementBase

	user            string `json:"user"`
	failIfNotExists bool   `json:"failIfNotExists"`
}

func NewDropUser(user string, failIfNotExists bool) *DropUser {
	rv := &DropUser{
		user:            user,
		failIfNotExists: failIfNotExists,
	}

	rv.stmt = rv
	return rv
}

func (this *DropUser) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropUser(this)
}

func (this *DropUser) Signature() value.Value {
	return nil
}

func (this *DropUser) Formalize() error {
	return nil
}

func (this *DropUser) MapExpressions(mapper expression.Mapper) (err error) {
	return nil
}

func (this *DropUser) Expressions() expression.Expressions {
	return nil
}

func (this *DropUser) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	// Currently our privileges always attach to buckets. In this case,
	// the data being updated isn't a bucket, it's system security data,
	// so the code is leaving the bucket name blank.
	// This works because no bucket name is needed for this type of authorization.
	// If we absolutely had to provide a table name, it would make sense to use system:user_info,
	// because that's the virtual table where the data can be accessed.
	privs.Add("", auth.PRIV_SECURITY_WRITE, auth.PRIV_PROPS_NONE)
	return privs, nil
}

func (this *DropUser) User() string {
	return this.user
}

func (this *DropUser) FailIfNotExists() bool {
	return this.failIfNotExists
}

func (this *DropUser) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "dropUser"}
	r["user"] = this.user
	r["failIfNotExists"] = this.failIfNotExists

	return json.Marshal(r)
}

func (this *DropUser) Type() string {
	return "DROP_USER"
}

func (this *DropUser) String() string {
	var s strings.Builder
	s.WriteString("DROP USER ")
	if !this.failIfNotExists {
		s.WriteString("IF EXISTS ")
	}
	s.WriteString(DecodeUsername(this.user))
	return s.String()
}
