//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build !enterprise
// +build !enterprise

package plan

import (
	"encoding/json"
)

type IndexAdvice struct {
	execution
}

func NewIndexAdvice() *IndexAdvice {
	return &IndexAdvice{}
}

func (this *IndexAdvice) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexAdvice(this)
}

func (this *IndexAdvice) New() Operator {
	return &IndexAdvice{}
}

func (this *IndexAdvice) Operator() Operator {
	return this
}

func (this *IndexAdvice) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *IndexAdvice) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "IndexAdvice"}

	if f != nil {
		f(r)
	}
	return r
}

func (this *IndexAdvice) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_ string `json:"#operator"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	return nil
}
