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
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/value"
)

// Create collection
type CreateCollection struct {
	ddl
	scope datastore.Scope
	node  *algebra.CreateCollection
}

func NewCreateCollection(scope datastore.Scope, node *algebra.CreateCollection) *CreateCollection {
	return &CreateCollection{
		scope: scope,
		node:  node,
	}
}

func (this *CreateCollection) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateCollection(this)
}

func (this *CreateCollection) New() Operator {
	return &CreateCollection{}
}

func (this *CreateCollection) Scope() datastore.Scope {
	return this.scope
}

func (this *CreateCollection) Node() *algebra.CreateCollection {
	return this.node
}

func (this *CreateCollection) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *CreateCollection) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "CreateCollection"}
	this.node.Keyspace().MarshalKeyspace(r)

	// invert so the default if not present is to fail if exists
	r["ifNotExists"] = !this.node.FailIfExists()

	r["with"] = this.node.With()

	if f != nil {
		f(r)
	}
	return r
}

func (this *CreateCollection) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string          `json:"#operator"`
		Namespace   string          `json:"namespace"`
		Bucket      string          `json:"bucket"`
		Scope       string          `json:"scope"`
		Keyspace    string          `json:"keyspace"`
		IfNotExists bool            `json:"ifNotExists"`
		With        json.RawMessage `json:"with"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	ksref := algebra.NewKeyspaceRefFromPath(algebra.NewPathLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope, _unmarshalled.Keyspace), "")
	this.scope, err = datastore.GetScope(ksref.Path().Parts()[0:3]...)
	if err != nil {
		return err
	}

	var with value.Value
	if len(_unmarshalled.With) > 0 {
		with = value.NewValue([]byte(_unmarshalled.With))
	}
	// invert IfNotExists to obtain FailIfExists
	this.node = algebra.NewCreateCollection(ksref, !_unmarshalled.IfNotExists, with)
	return nil
}

func (this *CreateCollection) verify(prepared *Prepared) bool {
	var res bool

	this.scope, res = verifyScope(this.scope, prepared)
	return res
}
