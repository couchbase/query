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
	"github.com/couchbase/query/errors"
)

// Drop collection
type DropCollection struct {
	ddl
	scope datastore.Scope
	node  *algebra.DropCollection
}

func NewDropCollection(scope datastore.Scope, node *algebra.DropCollection) *DropCollection {
	return &DropCollection{
		scope: scope,
		node:  node,
	}
}

func (this *DropCollection) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropCollection(this)
}

func (this *DropCollection) New() Operator {
	return &DropCollection{}
}

func (this *DropCollection) Scope() datastore.Scope {
	return this.scope
}

func (this *DropCollection) Node() *algebra.DropCollection {
	return this.node
}

func (this *DropCollection) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *DropCollection) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "DropCollection"}
	this.node.Keyspace().MarshalKeyspace(r)
	// invert so the default if not present is to fail if not exists
	r["ifExists"] = !this.node.FailIfNotExists()
	if f != nil {
		f(r)
	}
	return r
}

func (this *DropCollection) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string `json:"#operator"`
		Namespace string `json:"namespace"`
		Bucket    string `json:"bucket"`
		Scope     string `json:"scope"`
		Keyspace  string `json:"keyspace"`
		IfExists  bool   `json:"ifExists"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	// Build this.node.
	ksref := algebra.NewKeyspaceRefFromPath(algebra.NewPathLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope, _unmarshalled.Keyspace), "")
	this.scope, err = datastore.GetScope(ksref.Path().Parts()[0:3]...)
	if err != nil {
		return err
	}
	// invert IfExists to obtain FailIfNotExists
	this.node = algebra.NewDropCollection(ksref, !_unmarshalled.IfExists)

	return nil
}

func (this *DropCollection) verify(prepared *Prepared) errors.Error {
	var err errors.Error

	this.scope, err = verifyScope(this.scope, prepared)
	return err
}
