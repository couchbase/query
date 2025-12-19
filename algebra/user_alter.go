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

type AlterUser struct {
	statementBase

	user         string                `json:"user"`
	password_set bool                  `json:"password_set"`
	password     expression.Expression `json:"password"`
	groups_set   bool                  `json:"groups_set"`
	groups       []string              `json:"groups"`
	name_set     bool                  `json:"name_set"`
	name         string                `json:"name"`
}

func NewAlterUser(user string, password expression.Expression, name value.Value, groups value.Value) *AlterUser {
	rv := &AlterUser{
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

func (this *AlterUser) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAlterUser(this)
}

func (this *AlterUser) Signature() value.Value {
	return nil
}

func (this *AlterUser) Formalize() error {
	return nil
}

func (this *AlterUser) MapExpressions(mapper expression.Mapper) (err error) {
	return nil
}

func (this *AlterUser) Expressions() expression.Expressions {
	return nil
}

func (this *AlterUser) Privileges() (*auth.Privileges, errors.Error) {
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

func (this *AlterUser) User() string {
	return this.user
}

func (this *AlterUser) Password() (expression.Expression, bool) {
	return this.password, this.password_set
}

func (this *AlterUser) Name() (string, bool) {
	return this.name, this.name_set
}

func (this *AlterUser) Groups() ([]string, bool) {
	return this.groups, this.groups_set
}

func (this *AlterUser) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "alterUser"}
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

func (this *AlterUser) Type() string {
	return "ALTER_USER"
}

const _REDACT_TOKEN = "****"

func (this *AlterUser) String() string {
	var s strings.Builder
	s.WriteString("ALTER USER ")
	s.WriteString(DecodeUsername(this.user))
	if this.password_set {
		s.WriteString(" PASSWORD \"")
		if p := this.password.Value(); p != nil {
			s.WriteString(_REDACT_TOKEN)
		} else {
			s.WriteString(this.password.String())
		}
		s.WriteString("\"")
	}
	if this.name_set {
		s.WriteString(" WITH \"")
		s.WriteString(this.name)
		s.WriteString("\"")
	}
	if this.groups_set {
		s.WriteString(" GROUPS ")
		for i, g := range this.groups {
			if i > 0 {
				s.WriteString(", ")
			}
			s.WriteRune('`')
			s.WriteString(g)
			s.WriteRune('`')
		}
	}
	return s.String()
}

func DecodeUsername(user string) string {
	if i := strings.Index(user, ":"); i != -1 {
		return "`" + user[:i] + "`:`" + user[i+1:] + "`"
	}
	return "`" + user + "`"
}
