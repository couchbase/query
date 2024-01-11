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

type CreateUser struct {
	statementBase

	user         string                `json:"user"`
	password_set bool                  `json:"password_set"`
	password     expression.Expression `json:"password"`
	groups_set   bool                  `json:"groups_set"`
	groups       []string              `json:"groups"`
	name_set     bool                  `json:"name_set"`
	name         string                `json:"name"`
}

func NewCreateUser(user string, password expression.Expression, name value.Value, groups value.Value) *CreateUser {
	rv := &CreateUser{
		user: user,
	}
	if password != nil {
		rv.password_set = true
		rv.password = password
	}
	if name != nil {
		rv.name_set = true
		rv.name = name.ToString()
	}
	if groups != nil {
		rv.groups_set = true
		act := groups.Actual().([]interface{})
		rv.groups = make([]string, len(act))
		for i := range act {
			rv.groups[i] = value.NewValue(act[i]).ToString()
		}
	}

	rv.stmt = rv
	return rv
}

func (this *CreateUser) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateUser(this)
}

func (this *CreateUser) Signature() value.Value {
	return nil
}

func (this *CreateUser) Formalize() error {
	return nil
}

func (this *CreateUser) MapExpressions(mapper expression.Mapper) (err error) {
	return nil
}

func (this *CreateUser) Expressions() expression.Expressions {
	return nil
}

func (this *CreateUser) Privileges() (*auth.Privileges, errors.Error) {
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

func (this *CreateUser) User() string {
	return this.user
}

func (this *CreateUser) Password() (expression.Expression, bool) {
	return this.password, this.password_set
}

func (this *CreateUser) Name() (string, bool) {
	return this.name, this.name_set
}

func (this *CreateUser) Groups() ([]string, bool) {
	return this.groups, this.groups_set
}

func (this *CreateUser) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "createUser"}
	r["user"] = this.user
	r["password_set"] = this.password_set
	if this.password_set {
		r["password"] = this.password
	}
	r["groups_set"] = this.groups_set
	if this.groups_set {
		r["groups"] = this.groups
	}
	r["name_set"] = this.name_set
	if this.name_set {
		r["name"] = this.name
	}

	return json.Marshal(r)
}

func (this *CreateUser) Type() string {
	return "CREATE_USER"
}
