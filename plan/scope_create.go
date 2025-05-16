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

// Create scope
type CreateScope struct {
	ddl
	bucket datastore.Bucket
	node   *algebra.CreateScope
}

func NewCreateScope(bucket datastore.Bucket, node *algebra.CreateScope) *CreateScope {
	return &CreateScope{
		bucket: bucket,
		node:   node,
	}
}

func (this *CreateScope) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateScope(this)
}

func (this *CreateScope) New() Operator {
	return &CreateScope{}
}

func (this *CreateScope) Bucket() datastore.Bucket {
	return this.bucket
}

func (this *CreateScope) Node() *algebra.CreateScope {
	return this.node
}

func (this *CreateScope) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *CreateScope) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "CreateScope"}
	this.node.Scope().MarshalKeyspace(r)

	// invert so the default if not present is to fail if exists
	r["ifNotExists"] = !this.node.FailIfExists()

	if f != nil {
		f(r)
	}
	return r
}

func (this *CreateScope) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string `json:"#operator"`
		Namespace   string `json:"namespace"`
		Bucket      string `json:"bucket"`
		Scope       string `json:"scope"`
		IfNotExists bool   `json:"ifNotExists"`
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

	// invert IfNotExists to obtain FailIfExists
	this.node = algebra.NewCreateScope(scpref, !_unmarshalled.IfNotExists)
	return nil
}

func (this *CreateScope) verify(prepared *Prepared) errors.Error {
	var err errors.Error

	this.bucket, err = verifyBucket(this.bucket, prepared)
	return err
}
