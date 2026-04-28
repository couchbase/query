//  Copyright 2026-Present Couchbase, Inc.
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

// Alter catalog
type AlterCatalog struct {
	ddl
	node *algebra.AlterCatalog
}

func NewAlterCatalog(node *algebra.AlterCatalog) *AlterCatalog {
	return &AlterCatalog{
		node: node,
	}
}

func (this *AlterCatalog) Accept(visitor Visitor) (any, error) {
	return visitor.VisitAlterCatalog(this)
}

func (this *AlterCatalog) New() Operator {
	return &AlterCatalog{}
}

func (this *AlterCatalog) Node() *algebra.AlterCatalog {
	return this.node
}

func (this *AlterCatalog) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *AlterCatalog) MarshalBase(f func(map[string]any)) map[string]any {
	r := map[string]any{"#operator": "AlterCatalog"}
	r["name"] = this.node.Name()
	if this.node.With() != nil {
		r["with"] = this.node.With()
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *AlterCatalog) UnmarshalJSON(body []byte) error {
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

	this.node = algebra.NewAlterCatalog(_unmarshalled.Name, with)
	return nil
}
