//  Copyright 2014-Present Couchbase, Inc.
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
	"github.com/couchbase/query/value"
)

// Create index
type InferKeyspace struct {
	execution
	keyspace datastore.Keyspace
	node     *algebra.InferKeyspace
}

func NewInferKeyspace(keyspace datastore.Keyspace, node *algebra.InferKeyspace) *InferKeyspace {
	return &InferKeyspace{
		keyspace: keyspace,
		node:     node,
	}
}

func (this *InferKeyspace) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitInferKeyspace(this)
}

func (this *InferKeyspace) New() Operator {
	return &InferKeyspace{}
}

func (this *InferKeyspace) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *InferKeyspace) Node() *algebra.InferKeyspace {
	return this.node
}

func (this *InferKeyspace) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *InferKeyspace) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "InferKeyspace"}
	this.node.Keyspace().MarshalKeyspace(r)
	r["using"] = this.node.Using()

	if this.node.With() != nil {
		r["with"] = this.node.With()
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *InferKeyspace) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string                  `json:"#operator"`
		Namespace string                  `json:"namespace"`
		Bucket    string                  `json:"bucket"`
		Scope     string                  `json:"scope"`
		Keyspace  string                  `json:"keyspace"`
		Using     datastore.InferenceType `json:"using"`
		With      json.RawMessage         `json:"with"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	ksref := algebra.NewKeyspaceRefFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope, _unmarshalled.Keyspace), "")
	this.keyspace, err = datastore.GetKeyspace(ksref.Path().Parts()...)
	if err != nil {
		return err
	}

	var with value.Value
	if len(_unmarshalled.With) > 0 {
		with = value.NewValue([]byte(_unmarshalled.With))
	}

	this.node = algebra.NewInferKeyspace(ksref, _unmarshalled.Using, with)
	return nil
}

func (this *InferKeyspace) verify(prepared *Prepared) errors.Error {
	var err errors.Error

	this.keyspace, err = verifyKeyspace(this.keyspace, prepared)
	return err
}

func (this *InferKeyspace) keyspaceReferences(prepared *Prepared) {
	prepared.addKeyspaceReference(this.keyspace)
}
