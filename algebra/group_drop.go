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

type DropGroup struct {
	statementBase

	group string `json:"group"`
}

func NewDropGroup(group string) *DropGroup {
	rv := &DropGroup{
		group: group,
	}

	rv.stmt = rv
	return rv
}

func (this *DropGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropGroup(this)
}

func (this *DropGroup) Signature() value.Value {
	return nil
}

func (this *DropGroup) Formalize() error {
	return nil
}

func (this *DropGroup) MapExpressions(mapper expression.Mapper) (err error) {
	return nil
}

func (this *DropGroup) Expressions() expression.Expressions {
	return nil
}

func (this *DropGroup) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	// This works because no bucket name is needed for this type of authorization.
	privs.Add("", auth.PRIV_SECURITY_WRITE, auth.PRIV_PROPS_NONE)
	return privs, nil
}

func (this *DropGroup) Group() string {
	return this.group
}

func (this *DropGroup) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "dropGroup"}
	r["group"] = this.group

	return json.Marshal(r)
}

func (this *DropGroup) Type() string {
	return "DROP_GROUP"
}
