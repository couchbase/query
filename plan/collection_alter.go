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
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/value"
)

// Alter collection
type AlterCollection struct {
	ddl
	scope datastore.Scope
	node  *algebra.AlterCollection
}

func NewAlterCollection(scope datastore.Scope, node *algebra.AlterCollection) *AlterCollection {
	return &AlterCollection{
		scope: scope,
		node:  node,
	}
}

func (this *AlterCollection) Accept(visitor Visitor) (any, error) {
	return visitor.VisitAlterCollection(this)
}

func (this *AlterCollection) New() Operator {
	return &AlterCollection{}
}

func (this *AlterCollection) Scope() datastore.Scope {
	return this.scope
}

func (this *AlterCollection) Node() *algebra.AlterCollection {
	return this.node
}

func (this *AlterCollection) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *AlterCollection) MarshalBase(f func(map[string]any)) map[string]any {
	r := map[string]any{"#operator": "AlterCollection"}
	this.node.Keyspace().MarshalKeyspace(r)
	if this.node.With() != nil {
		r["with"] = this.node.With()
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *AlterCollection) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string          `json:"#operator"`
		Namespace string          `json:"namespace"`
		Bucket    string          `json:"bucket"`
		Scope     string          `json:"scope"`
		Keyspace  string          `json:"keyspace"`
		With      json.RawMessage `json:"with"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	var with value.Value
	if len(_unmarshalled.With) > 0 {
		with = value.NewValue([]byte(_unmarshalled.With))
	}

	// Parse keyspace path
	ksref := algebra.NewKeyspaceRefFromPath(algebra.NewPathLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope, _unmarshalled.Keyspace), "")
	this.scope, err = datastore.GetScope(ksref.Path().Parts()[0:3]...)
	if err != nil {
		return err
	}

	this.node = algebra.NewAlterCollection(ksref, with)
	return nil
}
