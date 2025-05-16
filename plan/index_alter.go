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

// Alter index
type AlterIndex struct {
	ddl
	index         datastore.Index
	deferredError errors.Error
	indexer       datastore.Indexer
	node          *algebra.AlterIndex
	keyspace      datastore.Keyspace
}

func NewAlterIndex(index datastore.Index, err errors.Error, indexer datastore.Indexer, node *algebra.AlterIndex,
	keyspace datastore.Keyspace) *AlterIndex {
	return &AlterIndex{
		index:         index,
		deferredError: err,
		indexer:       indexer,
		node:          node,
		keyspace:      keyspace,
	}
}

func (this *AlterIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAlterIndex(this)
}

func (this *AlterIndex) New() Operator {
	return &AlterIndex{}
}

func (this *AlterIndex) Index() datastore.Index {
	return this.index
}

func (this *AlterIndex) DeferredError() errors.Error {
	return this.deferredError
}

func (this *AlterIndex) Node() *algebra.AlterIndex {
	return this.node
}

func (this *AlterIndex) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *AlterIndex) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *AlterIndex) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "AlterIndex"}
	if this.index != nil {
		r["index"] = this.index.Name()
		r["index_id"] = this.index.Id()
	} else {
		r["index"] = this.node.Name()
	}
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

func (this *AlterIndex) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string              `json:"#operator"`
		Index     string              `json:"index"`
		IndexId   string              `json:"index_id"`
		Namespace string              `json:"namespace"`
		Bucket    string              `json:"bucket"`
		Scope     string              `json:"scope"`
		Keyspace  string              `json:"keyspace"`
		Using     datastore.IndexType `json:"using"`
		With      json.RawMessage     `json:"with"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	// Build the node
	// Get the keyspace ref (namespace:keyspace)
	ksref := algebra.NewKeyspaceRefFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope, _unmarshalled.Keyspace), "")

	// Get the with clause
	var with value.Value
	if len(_unmarshalled.With) > 0 {
		with = value.NewValue([]byte(_unmarshalled.With))
	}

	this.node = algebra.NewAlterIndex(ksref, _unmarshalled.Index, _unmarshalled.Using, with)

	// Build the index
	this.keyspace, err = datastore.GetKeyspace(ksref.Path().Parts()...)
	if err != nil {
		return err
	}

	// Alter Index is only supported by GSI and doesnt support a USING clause
	indexer, err := this.keyspace.Indexer(_unmarshalled.Using)
	if err != nil {
		return err
	}

	var index datastore.Index
	if len(_unmarshalled.IndexId) > 0 {
		index, err = indexer.IndexById(_unmarshalled.IndexId)
		if err != nil {
			return err
		}
	} else {
		index, err = indexer.IndexByName(_unmarshalled.Index)
		if err != nil {
			return err
		}
	}

	if _, ok := index.(datastore.Index3); !ok {
		return errors.NewAlterIndexError()
	}

	this.index = index
	this.indexer = indexer

	return nil
}

func (this *AlterIndex) verify(prepared *Prepared) errors.Error {
	if this.index == nil {
		this.index, this.deferredError = this.indexer.IndexByName(this.node.Name())
		if this.deferredError != nil {
			return this.deferredError
		}
	}
	return verifyIndex(this.index, this.indexer, nil, prepared)
}
