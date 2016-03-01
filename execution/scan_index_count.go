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
	"time"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type IndexCountScan struct {
	base
	plan *plan.IndexCountScan
}

func NewIndexCountScan(plan *plan.IndexCountScan) *IndexCountScan {
	rv := &IndexCountScan{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *IndexCountScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexCountScan(this)
}

func (this *IndexCountScan) Copy() Operator {
	return &IndexCountScan{base: this.base.copy(), plan: this.plan}
}

func (this *IndexCountScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover()       // Recover from any panic
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		keyspaceTerm := this.plan.Term()
		scanVector := context.ScanVectorSource().ScanVector(keyspaceTerm.Namespace(), keyspaceTerm.Keyspace())
		var duration time.Duration
		timer := time.Now()
		defer context.AddPhaseTime("IndexCountScan", time.Since(timer)-duration)
		var count int64
		for _, span := range this.plan.Spans() {
			dspan, err := evalSpan(span, context)
			if err != nil {
				context.Error(errors.NewEvaluationError(err, "span"))
				return
			}
			subcount, err := this.plan.Index().Count(dspan, context.ScanConsistency(), scanVector)
			if err != nil {
				context.Error(errors.NewEvaluationError(err, "Count()"))
				return
			}
			count = count + subcount
		}
		this.sendItem(value.NewAnnotatedValue(count))
	})
}
