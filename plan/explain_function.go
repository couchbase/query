//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import (
	"encoding/json"

	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/functions/storage"
)

type ExplainFunction struct {
	execution

	funcName functions.FunctionName
}

func NewExplainFunction(funcName functions.FunctionName) *ExplainFunction {
	return &ExplainFunction{
		funcName: funcName,
	}
}

func (this *ExplainFunction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExplainFunction(this)
}

func (this *ExplainFunction) New() Operator {
	return &ExplainFunction{}
}

func (this *ExplainFunction) MarshalJSON() ([]byte, error) {
	rv, err := json.Marshal(this.MarshalBase(nil))
	return rv, err
}

func (this *ExplainFunction) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	rv := map[string]interface{}{"#operator": "ExplainFunction"}

	identity := make(map[string]interface{})
	this.funcName.Signature(identity)

	rv["identity"] = identity

	if f != nil {
		f(rv)
	}
	return rv
}

func (this *ExplainFunction) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_        string          `json:"#operator"`
		Identity json.RawMessage `json:"identity"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.funcName, err = storage.MakeName(_unmarshalled.Identity)

	if err != nil {
		return err
	}
	return nil
}

func (this *ExplainFunction) FuncName() functions.FunctionName {
	return this.funcName
}
