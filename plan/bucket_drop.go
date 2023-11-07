//  Copyright 2020-Present Couchbase, Inc.
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

// Drop scope
type DropBucket struct {
	ddl
	node *algebra.DropBucket
}

func NewDropBucket(node *algebra.DropBucket) *DropBucket {
	return &DropBucket{
		node: node,
	}
}

func (this *DropBucket) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropBucket(this)
}

func (this *DropBucket) New() Operator {
	return &DropBucket{}
}

func (this *DropBucket) Node() *algebra.DropBucket {
	return this.node
}

func (this *DropBucket) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *DropBucket) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "DropBucket"}
	r["name"] = this.node.Name()
	// invert so the default if not present is to fail if not exists
	r["ifExists"] = !this.node.FailIfNotExists()
	if f != nil {
		f(r)
	}
	return r
}

func (this *DropBucket) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_        string `json:"#operator"`
		Name     string `json:"namespace"`
		IfExists bool   `json:"ifExists"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	// invert IfExists to obtain FailIfNotExists
	this.node = algebra.NewDropBucket(_unmarshalled.Name, !_unmarshalled.IfExists)

	return nil
}
