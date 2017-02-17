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

type IndexCountScan2 struct {
	base
	plan         *plan.IndexCountScan2
	childChannel chan int64
}

func NewIndexCountScan2(plan *plan.IndexCountScan2, context *Context) *IndexCountScan2 {
	rv := &IndexCountScan2{
		base: newBase(context),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *IndexCountScan2) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexCountScan2(this)
}

func (this *IndexCountScan2) Copy() Operator {
	return &IndexCountScan2{base: this.base.copy(), plan: this.plan}
}

func (this *IndexCountScan2) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover() // Recover from any panic
		this.switchPhase(_EXECTIME)
		this.setExecPhase(INDEX_COUNT, context)
		defer func() { this.switchPhase(_NOTIME) }() // accrue current phase's time
		defer close(this.itemChannel)                // Broadcast that I have stopped
		defer this.notify()                          // Notify that I have stopped

		var count int64

		keyspaceTerm := this.plan.Term()
		scanVector := context.ScanVectorSource().ScanVector(keyspaceTerm.Namespace(), keyspaceTerm.Keyspace())
		dspans, empty, err := evalSpan2(this.plan.Spans(), context)
		if err == nil && !empty {
			count, err = this.plan.Index().Count2(context.RequestId(), dspans, this.plan.Distinct(), context.ScanConsistency(), scanVector)
		}

		if err != nil {
			context.Error(errors.NewEvaluationError(err, "scanCount()"))
		}

		av := value.NewAnnotatedValue(count)
		av.InheritCovers(parent)
		this.sendItem(av)
	})
}

func (this *IndexCountScan2) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
