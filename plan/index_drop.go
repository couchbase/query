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
)

// Drop index
type DropIndex struct {
	ddl
	index         datastore.Index
	deferredError errors.Error
	indexer       datastore.Indexer
	node          *algebra.DropIndex
}

func NewDropIndex(index datastore.Index, err errors.Error, indexer datastore.Indexer, node *algebra.DropIndex) *DropIndex {
	return &DropIndex{
		index:         index,
		deferredError: err,
		indexer:       indexer,
		node:          node,
	}
}

func (this *DropIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropIndex(this)
}

func (this *DropIndex) New() Operator {
	return &DropIndex{}
}

func (this *DropIndex) Index() datastore.Index {
	return this.index
}

func (this *DropIndex) DeferredError() errors.Error {
	return this.deferredError
}

func (this *DropIndex) Node() *algebra.DropIndex {
	return this.node
}

func (this *DropIndex) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *DropIndex) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "DropIndex"}
	this.node.Keyspace().MarshalKeyspace(r)
	r["using"] = this.node.Using()
	r["name"] = this.node.Name()
	if this.index != nil {
		r["index_id"] = this.index.Id()
	}
	if f != nil {
		f(r)
	}
	// invert so the default if not present is to fail if not exists
	r["ifExists"] = !this.node.FailIfNotExists()
	r["primaryOnly"] = this.node.PrimaryOnly()
	r["vector"] = this.node.Vector()
	return r
}

func (this *DropIndex) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string              `json:"#operator"`
		Namespace   string              `json:"namespace"`
		Bucket      string              `json:"bucket"`
		Scope       string              `json:"scope"`
		Keyspace    string              `json:"keyspace"`
		Using       datastore.IndexType `json:"using"`
		Name        string              `json:"name"`
		IndexId     string              `json:"index_id"`
		IfExists    bool                `json:"ifExists"`
		PrimaryOnly bool                `json:"primaryOnly"`
		Vector      bool                `json:"vector"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	// Build this.node.
	ksref := algebra.NewKeyspaceRefFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope, _unmarshalled.Keyspace), "")
	// invert IfExists to obtain FailIfNotExists
	this.node = algebra.NewDropIndex(ksref, _unmarshalled.Name, _unmarshalled.Using, !_unmarshalled.IfExists,
		_unmarshalled.PrimaryOnly, _unmarshalled.Vector)

	// Build this.index.
	keyspace, err := datastore.GetKeyspace(ksref.Path().Parts()...)
	if err != nil {
		return err
	}
	indexer, err := keyspace.Indexer(_unmarshalled.Using)
	if err != nil {
		return err
	}
	if len(_unmarshalled.IndexId) > 0 {
		this.index, this.deferredError = indexer.IndexById(_unmarshalled.IndexId)
		if this.deferredError != nil {
			return this.deferredError
		}
	} else {
		this.index, this.deferredError = indexer.IndexByName(_unmarshalled.Name)
		if this.deferredError != nil {
			return this.deferredError
		}
	}
	this.indexer = indexer

	return nil
}

func (this *DropIndex) verify(prepared *Prepared) bool {
	if this.index == nil {
		this.index, this.deferredError = this.indexer.IndexByName(this.node.Name())
		if this.deferredError != nil {
			return false
		}
	}
	return verifyIndex(this.index, this.indexer, nil, prepared)
}
