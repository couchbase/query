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

// Flush collection
type FlushCollection struct {
	ddl
	keyspace datastore.Keyspace
	node     *algebra.FlushCollection
}

func NewFlushCollection(keyspace datastore.Keyspace, node *algebra.FlushCollection) *FlushCollection {
	return &FlushCollection{
		keyspace: keyspace,
		node:     node,
	}
}

func (this *FlushCollection) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFlushCollection(this)
}

func (this *FlushCollection) New() Operator {
	return &FlushCollection{}
}

func (this *FlushCollection) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *FlushCollection) Node() *algebra.FlushCollection {
	return this.node
}

func (this *FlushCollection) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *FlushCollection) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "FlushCollection"}
	this.node.Keyspace().MarshalKeyspace(r)

	if f != nil {
		f(r)
	}
	return r
}

func (this *FlushCollection) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string `json:"#operator"`
		Namespace string `json:"namespace"`
		Bucket    string `json:"bucket"`
		Scope     string `json:"scope"`
		Keyspace  string `json:"keyspace"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	ksref := algebra.NewKeyspaceRefFromPath(algebra.NewPathLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope, _unmarshalled.Keyspace), "")
	this.keyspace, err = datastore.GetKeyspace(ksref.Path().Parts()...)
	if err != nil {
		return err
	}

	this.node = algebra.NewFlushCollection(ksref)
	return nil
}

func (this *FlushCollection) verify(prepared *Prepared) errors.Error {
	var err errors.Error

	this.keyspace, err = verifyKeyspace(this.keyspace, prepared)
	return err
}
