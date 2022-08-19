//  Copyright 2019-Present Couchbase, Inc.
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
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/functions/storage"
)

// Drop function
type DropFunction struct {
	ddl
	name            functions.FunctionName
	failIfNotExists bool
}

func NewDropFunction(node *algebra.DropFunction) *DropFunction {
	return &DropFunction{
		name:            node.Name(),
		failIfNotExists: node.FailIfNotExists(),
	}
}

func (this *DropFunction) Name() functions.FunctionName {
	return this.name
}

func (this *DropFunction) FailIfNotExists() bool {
	return this.failIfNotExists
}

func (this *DropFunction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropFunction(this)
}

func (this *DropFunction) New() Operator {
	return &DropFunction{}
}

func (this *DropFunction) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *DropFunction) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "DropFunction"}
	identity := make(map[string]interface{})
	this.name.Signature(identity)
	r["identity"] = identity
	r["fail_if_not_exists"] = this.failIfNotExists

	if f != nil {
		f(r)
	}
	return r
}

func (this *DropFunction) UnmarshalJSON(bytes []byte) error {
	var _unmarshalled struct {
		_               string          `json:"#operator"`
		Identity        json.RawMessage `json:"identity"`
		FailIfNotExists bool            `json:"fail_if_not_exists"`
	}

	err := json.Unmarshal(bytes, &_unmarshalled)
	if err != nil {
		return err
	}

	this.name, err = storage.MakeName(_unmarshalled.Identity)
	if err != nil {
		return err
	}

	this.failIfNotExists = _unmarshalled.FailIfNotExists
	return nil
}
