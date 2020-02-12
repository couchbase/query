//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

// +build enterprise

package plan

import (
	"encoding/json"
	"github.com/couchbase/query-ee/indexadvisor/iaplan"
	"github.com/couchbase/query/expression"
)

type IndexAdvice struct {
	readonly
	adviceInfo *iaplan.IndexAdviceInfo
}

func NewIndexAdvice(queryInfos map[expression.HasExpressions]*iaplan.QueryInfo, coverIdxes iaplan.IndexInfos) *IndexAdvice {
	rv := &IndexAdvice{}
	cntKeyspaceNotFound := 0
	curIndexes := make(iaplan.IndexInfos, 0, 1) //initialize to distinguish between nil and empty for error message
	recIndexes := make(iaplan.IndexInfos, 0, 1)

	for _, v := range queryInfos {
		if !v.IsKeyspaceFound() {
			cntKeyspaceNotFound += 1
			continue
		}
		if len(v.GetCurIndexes()) > 0 {
			curIndexes = append(curIndexes, v.GetCurIndexes()...)
		}

		if len(v.GetUncoverIndexes()) > 0 {
			recIndexes = append(recIndexes, v.GetUncoverIndexes()...)
		}
	}

	if cntKeyspaceNotFound == len(queryInfos) && len(curIndexes) == 0 {
		curIndexes = nil
	}

	rv.adviceInfo = iaplan.NewIndexAdviceInfo(curIndexes, recIndexes, coverIdxes)
	return rv
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
