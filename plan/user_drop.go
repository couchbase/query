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
)

type DropUser struct {
	ddl
	node *algebra.DropUser
}

func NewDropUser(node *algebra.DropUser) *DropUser {
	return &DropUser{
		node: node,
	}
}

func (this *DropUser) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropUser(this)
}

func (this *DropUser) New() Operator {
	return &DropUser{}
}

func (this *DropUser) Node() *algebra.DropUser {
	return this.node
}

func (this *DropUser) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *DropUser) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "DropUser"}
	r["user"] = this.node.User()
	r["ifExists"] = !this.node.FailIfNotExists()
	if f != nil {
		f(r)
	}
	return r
}

func (this *DropUser) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_        string `json:"#operator"`
		User     string `json:"user"`
		IfExists bool   `json:"ifExists"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.node = algebra.NewDropUser(_unmarshalled.User, !_unmarshalled.IfExists)
	return nil
}
