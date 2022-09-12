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

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

type DropSequence struct {
	statementBase

	name            *Path `json:"name"`
	failIfNotExists bool  `json:"failIfNotExists"`
}

func NewDropSequence(name *Path, failIfNotExists bool) *DropSequence {
	rv := &DropSequence{
		name:            name,
		failIfNotExists: failIfNotExists,
	}

	rv.stmt = rv
	return rv
}

func (this *DropSequence) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropSequence(this)
}

func (this *DropSequence) Signature() value.Value {
	return nil
}

func (this *DropSequence) Formalize() error {
	return nil
}

func (this *DropSequence) MapExpressions(mapper expression.Mapper) error {
	return nil
}

func (this *DropSequence) Expressions() expression.Expressions {
	return nil
}

func (this *DropSequence) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	fullName := this.name.ScopePath().FullName()
	privs.Add(fullName, auth.PRIV_QUERY_MANAGE_SEQUENCES, auth.PRIV_PROPS_NONE)
	return privs, nil
}

func (this *DropSequence) Name() *Path {
	return this.name
}

func (this *DropSequence) FailIfNotExists() bool {
	return this.failIfNotExists
}

func (this *DropSequence) MarshalName(m map[string]interface{}) {
	m["namespace"] = this.name.Namespace()
	m["bucket"] = this.name.Bucket()
	m["scope"] = this.name.Scope()
	m["keyspace"] = this.name.Keyspace()
}
func (this *DropSequence) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "dropSequence"}
	this.MarshalName(r)
	r["failIfNotExists"] = this.failIfNotExists
	return json.Marshal(r)
}

func (this *DropSequence) Type() string {
	return "DROP_SEQUENCE"
}
