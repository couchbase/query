//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package plan

import (
	"encoding/json"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
)

// Drop index
type DropIndex struct {
	ddl
	index   datastore.Index
	indexer datastore.Indexer
	node    *algebra.DropIndex
}

func NewDropIndex(index datastore.Index, indexer datastore.Indexer, node *algebra.DropIndex) *DropIndex {
	return &DropIndex{
		index:   index,
		indexer: indexer,
		node:    node,
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
	r["index_id"] = this.index.Id()
	if f != nil {
		f(r)
	}
	return r
}

func (this *DropIndex) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string              `json:"#operator"`
		Namespace string              `json:"namespace"`
		Bucket    string              `json:"bucket"`
		Scope     string              `json:"scope"`
		Keyspace  string              `json:"keyspace"`
		Using     datastore.IndexType `json:"using"`
		Name      string              `json:"name"`
		IndexId   string              `json:"index_id"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	// Build this.node.
	ksref := algebra.NewKeyspaceRefFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope, _unmarshalled.Keyspace), "")
	this.node = algebra.NewDropIndex(ksref, _unmarshalled.Name, _unmarshalled.Using)

	// Build this.index.
	keyspace, err := datastore.GetKeyspace(ksref.Path().Parts()...)
	if err != nil {
		return err
	}
	indexer, err := keyspace.Indexer(_unmarshalled.Using)
	if err != nil {
		return err
	}
	index, err := indexer.IndexById(_unmarshalled.IndexId)
	if err != nil {
		return err
	}
	this.index = index
	this.indexer = indexer

	return nil
}

func (this *DropIndex) verify(prepared *Prepared) bool {
	return verifyIndex(this.index, this.indexer, nil, prepared)
}
