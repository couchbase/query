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

type AlterGroup struct {
	statementBase

	group     string   `json:"user"`
	desc_set  bool     `json:"desc_set"`
	desc      string   `json:"desc"`
	roles_set bool     `json:"roles_set"`
	roles     []string `json:"roles"`
}

func NewAlterGroup(group string, desc value.Value, roles value.Value) *AlterGroup {
	rv := &AlterGroup{
		group: group,
	}
	if desc != nil {
		rv.desc_set = true
		rv.desc = desc.ToString()
	}
	if roles != nil {
		rv.roles_set = true
		act := roles.Actual().([]interface{})
		rv.roles = make([]string, len(act))
		for i := range act {
			rv.roles[i] = value.NewValue(act[i]).ToString()
		}
	}
	rv.stmt = rv
	return rv
}

func (this *AlterGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAlterGroup(this)
}

func (this *AlterGroup) Signature() value.Value {
	return nil
}

func (this *AlterGroup) Formalize() error {
	return nil
}

func (this *AlterGroup) MapExpressions(mapper expression.Mapper) (err error) {
	return nil
}

func (this *AlterGroup) Expressions() expression.Expressions {
	return nil
}

func (this *AlterGroup) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	// This works because no bucket name is needed for this type of authorization.
	privs.Add("", auth.PRIV_SECURITY_WRITE, auth.PRIV_PROPS_NONE)
	return privs, nil
}

func (this *AlterGroup) Group() string {
	return this.group
}

func (this *AlterGroup) Desc() (string, bool) {
	return this.desc, this.desc_set
}

func (this *AlterGroup) Roles() ([]string, bool) {
	return this.roles, this.roles_set
}

func (this *AlterGroup) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "alterGroup"}
	r["group"] = this.group
	r["desc_set"] = this.desc_set
	if this.desc_set {
		r["desc"] = this.desc
	}
	r["roles_set"] = this.roles_set
	if this.roles_set {
		r["roles"] = this.roles
	}

	return json.Marshal(r)
}

func (this *AlterGroup) Type() string {
	return "ALTER_GROUP"
}

func (this *AlterGroup) String() string {
	var s strings.Builder
	s.WriteString("ALTER GROUP ")
	s.WriteRune('`')
	s.WriteString(this.group)
	s.WriteRune('`')
	if this.desc_set {
		s.WriteString(" WITH \"")
		s.WriteString(this.desc)
		s.WriteString("\"")
	}

	if this.roles_set {
		if len(this.roles) == 0 {
			s.WriteString(" NO ROLES")
		} else {
			s.WriteString(" ROLES ")
			for i, r := range this.roles {
				if i > 0 {
					s.WriteString(", ")
				}
				s.WriteString(DecodeParsedRole(r))
			}
		}
	}
	return s.String()
}

func DecodeParsedRole(role string) string {
	if i := strings.Index(role, "["); i != -1 {
		rolename := role[:i]
		if keyspace := role[i+1 : len(role)-1]; len(keyspace) > 0 {
			parts := strings.Split(keyspace, ":")
			keyspacename := ""
			l := len(parts) - 1
			for j, part := range parts {
				keyspacename += "`" + part + "`"
				if j != l {
					keyspacename += "."
				}
			}
			return "`" + auth.RoleToAlias(rolename) + "` ON " + keyspacename
		}
		return "`" + rolename + "`"
	}
	return "`" + role + "`"
}
