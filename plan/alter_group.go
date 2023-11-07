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

type AlterGroup struct {
	ddl
	node *algebra.AlterGroup
}

func NewAlterGroup(node *algebra.AlterGroup) *AlterGroup {
	return &AlterGroup{
		node: node,
	}
}

func (this *AlterGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAlterGroup(this)
}

func (this *AlterGroup) New() Operator {
	return &AlterGroup{}
}

func (this *AlterGroup) Node() *algebra.AlterGroup {
	return this.node
}

func (this *AlterGroup) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *AlterGroup) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "AlterGroup"}
	r["group"] = this.node.Group()
	desc, ok := this.node.Desc()
	r["desc_set"] = ok
	if ok {
		r["desc"] = desc
	}
	roles, ok := this.node.Roles()
	r["roles_set"] = ok
	if ok {
		r["roles"] = roles
	}
	if f != nil {
		f(r)
	}
	return r
}

func (this *AlterGroup) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_        string   `json:"#operator"`
		Group    string   `json:"group"`
		DescSet  bool     `json:"desc_set"`
		Desc     string   `json:"desc"`
		RolesSet bool     `json:"roles_set"`
		Roles    []string `json:"roles"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	var desc, roles value.Value
	if _unmarshalled.DescSet {
		desc = value.NewValue(_unmarshalled.Desc)
	}
	if _unmarshalled.RolesSet {
		roles = value.NewValue(_unmarshalled.Roles)
	}

	this.node = algebra.NewAlterGroup(_unmarshalled.Group, desc, roles)
	return nil
}
