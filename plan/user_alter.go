//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import (
	"encoding/json"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/parser/n1ql"
	"github.com/couchbase/query/value"
)

type AlterUser struct {
	ddl
	node *algebra.AlterUser
}

func NewAlterUser(node *algebra.AlterUser) *AlterUser {
	return &AlterUser{
		node: node,
	}
}

func (this *AlterUser) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAlterUser(this)
}

func (this *AlterUser) New() Operator {
	return &AlterUser{}
}

func (this *AlterUser) Node() *algebra.AlterUser {
	return this.node
}

func (this *AlterUser) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *AlterUser) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "AlterUser"}
	r["user"] = this.node.User()
	password, ok := this.node.Password()
	r["password_set"] = ok
	if ok {
		r["password"] = password
	}
	groups, ok := this.node.Groups()
	r["groups_set"] = ok
	if ok {
		r["groups"] = groups
	}
	name, ok := this.node.Name()
	r["name_set"] = ok
	if ok {
		r["name"] = name
	}
	if f != nil {
		f(r)
	}
	return r
}

func (this *AlterUser) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string   `json:"#operator"`
		User        string   `json:"user"`
		PasswordSet bool     `json:"password_set"`
		Password    string   `json:"password"`
		GroupsSet   bool     `json:"groups_set"`
		Groups      []string `json:"groups"`
		NameSet     bool     `json:"name_set"`
		Name        string   `json:"name"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	var groups, name value.Value
	var password expression.Expression

	if _unmarshalled.PasswordSet {
		password, err = n1ql.ParseExpression(_unmarshalled.Password)
		if err != nil {
			return err
		}
	}
	if _unmarshalled.NameSet {
		name = value.NewValue(_unmarshalled.Name)
	}
	if _unmarshalled.GroupsSet {
		groups = value.NewValue(_unmarshalled.Groups)
	}

	this.node = algebra.NewAlterUser(_unmarshalled.User, password, name, groups)
	return nil
}
