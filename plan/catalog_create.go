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

// Create catalog
type CreateCatalog struct {
	ddl
	node *algebra.CreateCatalog
}

func NewCreateCatalog(node *algebra.CreateCatalog) *CreateCatalog {
	return &CreateCatalog{
		node: node,
	}
}

func (this *CreateCatalog) Accept(visitor Visitor) (any, error) {
	return visitor.VisitCreateCatalog(this)
}

func (this *CreateCatalog) New() Operator {
	return &CreateCatalog{}
}

func (this *CreateCatalog) Node() *algebra.CreateCatalog {
	return this.node
}

func (this *CreateCatalog) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *CreateCatalog) MarshalBase(f func(map[string]any)) map[string]any {
	r := map[string]any{"#operator": "CreateCatalog"}
	r["name"] = this.node.Name()
	r["catalogType"] = this.node.CatalogType()
	r["source"] = this.node.Source()
	r["credential"] = this.node.Credential()
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

func (this *CreateCatalog) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string          `json:"#operator"`
		Name        string          `json:"name"`
		CatalogType string          `json:"catalogType"`
		Source      string          `json:"source"`
		Credential  string          `json:"credential"`
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
	this.node = algebra.NewCreateCatalog(_unmarshalled.Name, _unmarshalled.CatalogType,
		_unmarshalled.Source, _unmarshalled.Credential, !_unmarshalled.IfNotExists, with)
	return nil
}
