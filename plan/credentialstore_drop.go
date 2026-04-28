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

type DropCredentialStore struct {
	ddl
	node *algebra.DropCredentialStore
}

func NewDropCredentialStore(node *algebra.DropCredentialStore) *DropCredentialStore {
	return &DropCredentialStore{
		node: node,
	}
}

func (this *DropCredentialStore) Accept(visitor Visitor) (any, error) {
	return visitor.VisitDropCredentialStore(this)
}

func (this *DropCredentialStore) New() Operator {
	return &DropCredentialStore{}
}

func (this *DropCredentialStore) Node() *algebra.DropCredentialStore {
	return this.node
}

func (this *DropCredentialStore) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *DropCredentialStore) MarshalBase(f func(map[string]any)) map[string]any {
	r := map[string]any{"#operator": "DropCredentialStore"}
	r["name"] = this.node.Name()
	// invert so the default if not present is to fail if not exists
	r["ifExists"] = !this.node.FailIfNotExists()
	if f != nil {
		f(r)
	}
	return r
}

func (this *DropCredentialStore) UnmarshalJSON(body []byte) error {
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
	this.node = algebra.NewDropCredentialStore(_unmarshalled.Name, !_unmarshalled.IfExists)
	return nil
}
