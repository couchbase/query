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
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/functions/storage"
)

// Execute function
type ExecuteFunction struct {
	ddl
	name  functions.FunctionName
	exprs expression.Expressions
}

func NewExecuteFunction(node *algebra.ExecuteFunction) *ExecuteFunction {
	return &ExecuteFunction{
		name:  node.Name(),
		exprs: node.Expressions(),
	}
}

func (this *ExecuteFunction) Name() functions.FunctionName {
	return this.name
}

func (this *ExecuteFunction) Expressions() expression.Expressions {
	return this.exprs
}

func (this *ExecuteFunction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExecuteFunction(this)
}

func (this *ExecuteFunction) New() Operator {
	return &ExecuteFunction{}
}

func (this *ExecuteFunction) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *ExecuteFunction) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "ExecuteFunction"}
	identity := make(map[string]interface{})
	this.name.Signature(identity)
	r["identity"] = identity

	if f != nil {
		f(r)
	}
	return r
}

func (this *ExecuteFunction) UnmarshalJSON(bytes []byte) error {
	var _unmarshalled struct {
		_        string          `json:"#operator"`
		Identity json.RawMessage `json:"identity"`
	}

	err := json.Unmarshal(bytes, &_unmarshalled)
	if err != nil {
		return err
	}

	this.name, err = storage.MakeName(_unmarshalled.Identity)
	if err != nil {
		return err
	}
	return nil
}
