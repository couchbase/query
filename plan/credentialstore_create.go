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

type CreateCredentialStore struct {
	ddl
	node *algebra.CreateCredentialStore
}

func NewCreateCredentialStore(node *algebra.CreateCredentialStore) *CreateCredentialStore {
	return &CreateCredentialStore{
		node: node,
	}
}

func (this *CreateCredentialStore) Accept(visitor Visitor) (any, error) {
	return visitor.VisitCreateCredentialStore(this)
}

func (this *CreateCredentialStore) New() Operator {
	return &CreateCredentialStore{}
}

func (this *CreateCredentialStore) Node() *algebra.CreateCredentialStore {
	return this.node
}

func (this *CreateCredentialStore) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *CreateCredentialStore) MarshalBase(f func(map[string]any)) map[string]any {
	r := map[string]any{"#operator": "CreateCredentialStore"}
	r["name"] = this.node.Name()
	// invert so the default if not present is to fail if exists
	r["ifNotExists"] = !this.node.FailIfExists()
	r["with"] = this.node.With()
	if f != nil {
		f(r)
	}
	return r
}

func (this *CreateCredentialStore) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string          `json:"#operator"`
		Name        string          `json:"name"`
		IfNotExists bool            `json:"ifNotExists"`
		With        json.RawMessage `json:"with"`
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
	this.node = algebra.NewCreateCredentialStore(_unmarshalled.Name, !_unmarshalled.IfNotExists, with)
	return nil
}
