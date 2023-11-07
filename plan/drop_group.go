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

type DropGroup struct {
	ddl
	node *algebra.DropGroup
}

func NewDropGroup(node *algebra.DropGroup) *DropGroup {
	return &DropGroup{
		node: node,
	}
}

func (this *DropGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropGroup(this)
}

func (this *DropGroup) New() Operator {
	return &DropGroup{}
}

func (this *DropGroup) Node() *algebra.DropGroup {
	return this.node
}

func (this *DropGroup) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *DropGroup) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "DropGroup"}
	r["group"] = this.node.Group()
	if f != nil {
		f(r)
	}
	return r
}

func (this *DropGroup) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string `json:"#operator"`
		Group string `json:"group"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.node = algebra.NewDropGroup(_unmarshalled.Group)
	return nil
}
