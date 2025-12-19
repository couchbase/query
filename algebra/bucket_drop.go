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

type DropBucket struct {
	statementBase

	name            string `json:"name"`
	failIfNotExists bool   `json:"failIfNotExists"`
}

func NewDropBucket(name string, failIfNotExists bool) *DropBucket {
	rv := &DropBucket{
		name:            name,
		failIfNotExists: failIfNotExists,
	}

	rv.stmt = rv
	return rv
}

func (this *DropBucket) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropBucket(this)
}

func (this *DropBucket) Signature() value.Value {
	return nil
}

func (this *DropBucket) Formalize() error {
	return nil
}

func (this *DropBucket) MapExpressions(mapper expression.Mapper) error {
	return nil
}

func (this *DropBucket) Expressions() expression.Expressions {
	return nil
}

func (this *DropBucket) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	privs.Add("", auth.PRIV_CLUSTER_ADMIN, auth.PRIV_PROPS_NONE)
	return privs, nil
}

func (this *DropBucket) Name() string {
	return this.name
}

func (this *DropBucket) FailIfNotExists() bool {
	return this.failIfNotExists
}

func (this *DropBucket) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "dropBucket"}
	r["name"] = this.name
	r["failIfNotExists"] = this.failIfNotExists
	return json.Marshal(r)
}

func (this *DropBucket) Type() string {
	return "DROP_BUCKET"
}

func (this *DropBucket) String() string {
	var s strings.Builder
	s.WriteString("DROP BUCKET ")

	if !this.failIfNotExists {
		s.WriteString("IF EXISTS ")
	}

	s.WriteRune('`')
	s.WriteString(this.name)
	s.WriteRune('`')
	return s.String()
}
