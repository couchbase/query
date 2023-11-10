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
	"github.com/couchbase/query/value"
)

type CreateUser struct {
	ddl
	node *algebra.CreateUser
}

func NewCreateUser(node *algebra.CreateUser) *CreateUser {
	return &CreateUser{
		node: node,
	}
}

func (this *CreateUser) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateUser(this)
}

func (this *CreateUser) New() Operator {
	return &CreateUser{}
}

func (this *CreateUser) Node() *algebra.CreateUser {
	return this.node
}

func (this *CreateUser) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *CreateUser) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "CreateUser"}
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

func (this *CreateUser) UnmarshalJSON(body []byte) error {
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

	var password, groups, name value.Value
	if _unmarshalled.PasswordSet {
		password = value.NewValue(_unmarshalled.Password)
	}
	if _unmarshalled.NameSet {
		name = value.NewValue(_unmarshalled.Name)
	}
	if _unmarshalled.GroupsSet {
		groups = value.NewValue(_unmarshalled.Groups)
	}

	this.node = algebra.NewCreateUser(_unmarshalled.User, password, name, groups)
	return nil
}
