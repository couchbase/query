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

// Drop scope
type DropScope struct {
	ddl
	bucket datastore.Bucket
	node   *algebra.DropScope
}

func NewDropScope(bucket datastore.Bucket, node *algebra.DropScope) *DropScope {
	return &DropScope{
		bucket: bucket,
		node:   node,
	}
}

func (this *DropScope) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropScope(this)
}

func (this *DropScope) New() Operator {
	return &DropScope{}
}

func (this *DropScope) Node() *algebra.DropScope {
	return this.node
}

func (this *DropScope) Bucket() datastore.Bucket {
	return this.bucket
}

func (this *DropScope) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *DropScope) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "DropScope"}
	this.node.Scope().MarshalKeyspace(r)
	// invert so the default if not present is to fail if not exists
	r["ifExists"] = !this.node.FailIfNotExists()
	if f != nil {
		f(r)
	}
	return r
}

func (this *DropScope) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string `json:"#operator"`
		Namespace string `json:"namespace"`
		Bucket    string `json:"bucket"`
		Scope     string `json:"scope"`
		IfExists  bool   `json:"ifExists"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	scpref := algebra.NewScopeRefFromPath(algebra.NewPathScope(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope), "")
	this.bucket, err = datastore.GetBucket(_unmarshalled.Namespace, _unmarshalled.Bucket)
	if err != nil {
		return err
	}

	// invert IfExists to obtain FailIfNotExists
	this.node = algebra.NewDropScope(scpref, !_unmarshalled.IfExists)

	return nil
}

func (this *DropScope) verify(prepared *Prepared) errors.Error {
	var err errors.Error

	this.bucket, err = verifyBucket(this.bucket, prepared)
	return err
}
