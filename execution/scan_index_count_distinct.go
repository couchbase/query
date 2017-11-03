//  Copyright (c) 2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execution

import (
	"encoding/json"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type IndexCountDistinctScan2 struct {
	base
	plan *plan.IndexCountDistinctScan2
}

func NewIndexCountDistinctScan2(plan *plan.IndexCountDistinctScan2, context *Context) *IndexCountDistinctScan2 {
	rv := &IndexCountDistinctScan2{
		plan: plan,
	}

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *IndexCountDistinctScan2) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexCountDistinctScan2(this)
}

func (this *IndexCountDistinctScan2) Copy() Operator {
	rv := &IndexCountDistinctScan2{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *IndexCountDistinctScan2) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover() // Recover from any panic
		this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		this.setExecPhase(INDEX_COUNT, context)
		defer func() { this.switchPhase(_NOTIME) }() // accrue current phase's time
		defer this.notify()                          // Notify that I have stopped

		var count int64

		keyspaceTerm := this.plan.Term()
		scanVector := context.ScanVectorSource().ScanVector(keyspaceTerm.Namespace(), keyspaceTerm.Keyspace())
		dspans, empty, err := evalSpan2(this.plan.Spans(), nil, context)
		if err == nil && !empty {
			count, err = this.plan.Index().CountDistinct(context.RequestId(), dspans, context.ScanConsistency(), scanVector)
		}

		if err != nil {
			context.Error(errors.NewEvaluationError(err, "scanCountDistinct()"))
		}

		av := value.NewAnnotatedValue(count)
		av.InheritCovers(parent)
		this.sendItem(av)
	})
}

func (this *IndexCountDistinctScan2) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
