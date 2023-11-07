//  Copyright 2014-Present Couchbase, Inc.
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
)

// Grant role
type GrantRole struct {
	ddl
	node *algebra.GrantRole
}

func NewGrantRole(node *algebra.GrantRole) *GrantRole {
	return &GrantRole{
		node: node,
	}
}

func (this *GrantRole) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitGrantRole(this)
}

func (this *GrantRole) New() Operator {
	return &GrantRole{}
}

func (this *GrantRole) Node() *algebra.GrantRole {
	return this.node
}

func (this *GrantRole) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *GrantRole) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "GrantRole"}
	r["roles"] = this.node.Roles()
	r["keyspaces"] = this.node.Keyspaces()
	r["users"] = this.node.Users()
	r["groups"] = this.node.Groups()
	if f != nil {
		f(r)
	}
	return r
}

func (this *GrantRole) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string                 `json:"#operator"`
		Roles     []string               `json:"roles"`
		Keyspaces []*algebra.KeyspaceRef `json:"keyspaces"`
		Users     []string               `json:"users"`
		Groups    bool                   `json:"groups"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.node = algebra.NewGrantRole(_unmarshalled.Roles, _unmarshalled.Keyspaces, _unmarshalled.Users, _unmarshalled.Groups)
	return nil
}

/*
Currently there's no need to verify role statements:
if a keyspace has been dropped and recreated, execution will succeed
if a keyspace has been dropped, reprepare will fail anyway

func (this *GrantRole) verify(prepared *Prepared) bool {
	for _, keyspace := range this.node.Keyspaces() {
		_, res := verifyKeyspaceName(keyspace, prepared)
		if !res {
			return false
		}
	}
	return true
}
*/
