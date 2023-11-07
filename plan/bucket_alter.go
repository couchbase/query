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

// Alter scope
type AlterBucket struct {
	ddl
	node *algebra.AlterBucket
}

func NewAlterBucket(node *algebra.AlterBucket) *AlterBucket {
	return &AlterBucket{
		node: node,
	}
}

func (this *AlterBucket) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAlterBucket(this)
}

func (this *AlterBucket) New() Operator {
	return &AlterBucket{}
}

func (this *AlterBucket) Node() *algebra.AlterBucket {
	return this.node
}

func (this *AlterBucket) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *AlterBucket) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "AlterBucket"}
	r["name"] = this.node.Name()
	r["with"] = this.node.With()

	if f != nil {
		f(r)
	}
	return r
}

func (this *AlterBucket) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_    string          `json:"#operator"`
		Name string          `json:"name"`
		With json.RawMessage `json:"with"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	var with value.Value
	if len(_unmarshalled.With) > 0 {
		with = value.NewValue([]byte(_unmarshalled.With))
	}

	this.node = algebra.NewAlterBucket(_unmarshalled.Name, with)
	return nil
}
