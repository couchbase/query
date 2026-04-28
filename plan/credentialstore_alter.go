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

type AlterCredentialStore struct {
	ddl
	node *algebra.AlterCredentialStore
}

func NewAlterCredentialStore(node *algebra.AlterCredentialStore) *AlterCredentialStore {
	return &AlterCredentialStore{
		node: node,
	}
}

func (this *AlterCredentialStore) Accept(visitor Visitor) (any, error) {
	return visitor.VisitAlterCredentialStore(this)
}

func (this *AlterCredentialStore) New() Operator {
	return &AlterCredentialStore{}
}

func (this *AlterCredentialStore) Node() *algebra.AlterCredentialStore {
	return this.node
}

func (this *AlterCredentialStore) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *AlterCredentialStore) MarshalBase(f func(map[string]any)) map[string]any {
	r := map[string]any{"#operator": "AlterCredentialStore"}
	r["name"] = this.node.Name()
	r["with"] = this.node.With()
	if f != nil {
		f(r)
	}
	return r
}

func (this *AlterCredentialStore) UnmarshalJSON(body []byte) error {
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
	this.node = algebra.NewAlterCredentialStore(_unmarshalled.Name, with)
	return nil
}
