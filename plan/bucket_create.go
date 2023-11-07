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

// Create scope
type CreateBucket struct {
	ddl
	node *algebra.CreateBucket
}

func NewCreateBucket(node *algebra.CreateBucket) *CreateBucket {
	return &CreateBucket{
		node: node,
	}
}

func (this *CreateBucket) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateBucket(this)
}

func (this *CreateBucket) New() Operator {
	return &CreateBucket{}
}

func (this *CreateBucket) Node() *algebra.CreateBucket {
	return this.node
}

func (this *CreateBucket) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *CreateBucket) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "CreateBucket"}
	r["name"] = this.node.Name()
	if this.node.With() != nil {
		r["with"] = this.node.With()
	}
	// invert so the default if not present is to fail if exists
	r["ifNotExists"] = !this.node.FailIfExists()

	if f != nil {
		f(r)
	}
	return r
}

func (this *CreateBucket) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string          `json:"#operator"`
		Name        string          `json:"name"`
		With        json.RawMessage `json:"with"`
		IfNotExists bool            `json:"ifNotExists"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	var with value.Value
	if len(_unmarshalled.With) > 0 {
		with = value.NewValue([]byte(_unmarshalled.With))
	}

	// invert IfNotExists to obtain FailIfExists
	this.node = algebra.NewCreateBucket(_unmarshalled.Name, !_unmarshalled.IfNotExists, with)
	return nil
}
