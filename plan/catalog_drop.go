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
)

// Drop catalog
type DropCatalog struct {
	ddl
	node *algebra.DropCatalog
}

func NewDropCatalog(node *algebra.DropCatalog) *DropCatalog {
	return &DropCatalog{
		node: node,
	}
}

func (this *DropCatalog) Accept(visitor Visitor) (any, error) {
	return visitor.VisitDropCatalog(this)
}

func (this *DropCatalog) New() Operator {
	return &DropCatalog{}
}

func (this *DropCatalog) Node() *algebra.DropCatalog {
	return this.node
}

func (this *DropCatalog) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *DropCatalog) MarshalBase(f func(map[string]any)) map[string]any {
	r := map[string]any{"#operator": "DropCatalog"}
	r["name"] = this.node.Name()
	// invert so the default if not present is to fail if not exists
	r["ifExists"] = !this.node.FailIfNotExists()

	if f != nil {
		f(r)
	}
	return r
}

func (this *DropCatalog) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_        string `json:"#operator"`
		Name     string `json:"name"`
		IfExists bool   `json:"ifExists"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	// invert IfExists to obtain FailIfNotExists
	this.node = algebra.NewDropCatalog(_unmarshalled.Name, !_unmarshalled.IfExists)
	return nil
}
