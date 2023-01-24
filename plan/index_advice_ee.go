//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build enterprise
// +build enterprise

package plan

import (
	"encoding/json"

	"github.com/couchbase/query-ee/indexadvisor/iaplan"
)

type IndexAdvice struct {
	execution
	adviceInfo *iaplan.IndexAdviceInfo
}

func NewIndexAdvice(curIndexes, recIndexes, coverIdxes iaplan.IndexInfos) *IndexAdvice {
	return &IndexAdvice{
		adviceInfo: iaplan.NewIndexAdviceInfo(curIndexes, recIndexes, coverIdxes),
	}
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
	r["adviseinfo"] = this.adviceInfo

	if f != nil {
		f(r)
	}
	return r
}

func (this *IndexAdvice) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_          string          `json:"#operator"`
		AdviceInfo json.RawMessage `json:"adviseinfo"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	if _unmarshalled.AdviceInfo != nil {
		r := &iaplan.IndexAdviceInfo{}
		err = r.UnmarshalJSON(_unmarshalled.AdviceInfo)
		if err != nil {
			return err
		}
		this.adviceInfo = r
	}
	return nil
}
