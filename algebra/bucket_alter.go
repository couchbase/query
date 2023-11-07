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

type AlterBucket struct {
	statementBase

	name string      `json:"name"`
	with value.Value `json:"with"`
}

func NewAlterBucket(name string, with value.Value) *AlterBucket {
	rv := &AlterBucket{
		name: name,
		with: with,
	}

	rv.stmt = rv
	return rv
}

func (this *AlterBucket) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAlterBucket(this)
}

func (this *AlterBucket) Signature() value.Value {
	return nil
}

func (this *AlterBucket) Formalize() error {
	return nil
}

func (this *AlterBucket) MapExpressions(mapper expression.Mapper) error {
	return nil
}

func (this *AlterBucket) Expressions() expression.Expressions {
	return nil
}

func (this *AlterBucket) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	privs.Add("", auth.PRIV_CLUSTER_ADMIN, auth.PRIV_PROPS_NONE)

	return privs, nil
}

func (this *AlterBucket) Name() string {
	return this.name
}

func (this *AlterBucket) With() value.Value {
	return this.with
}

func (this *AlterBucket) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "alterBucket"}
	r["name"] = this.name
	if this.with != nil {
		r["with"] = this.with
	}
	return json.Marshal(r)
}

func (this *AlterBucket) Type() string {
	return "ALTER_BUCKET"
}
