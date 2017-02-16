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
	"time"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type IndexCountScan struct {
	base
	plan         *plan.IndexCountScan
	childChannel chan int64
}

func NewIndexCountScan(plan *plan.IndexCountScan, context *Context) *IndexCountScan {
	rv := &IndexCountScan{
		base: newRedirectBase(),
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
		defer context.Recover() // Recover from any panic
		this.switchPhase(_EXECTIME)
		this.phaseTimes = func(d time.Duration) { context.AddPhaseTime(INDEX_COUNT, d) }
		defer func() { this.switchPhase(_NOTIME) }() // accrue current phase's time
		defer close(this.itemChannel)                // Broadcast that I have stopped
		defer this.notify()                          // Notify that I have stopped

		spans := this.plan.Spans()
		n := len(spans)
		this.childChannel = make(chan int64, n)
		keyspaceTerm := this.plan.Term()
		scanVector := context.ScanVectorSource().ScanVector(keyspaceTerm.Namespace(), keyspaceTerm.Keyspace())

		var count int64
		var subcount int64
		for _, span := range spans {
			go this.scanCount(span, scanVector, context)
		}

		for n > 0 {
			this.switchPhase(_SERVTIME)
			select {
			case <-this.stopChannel:
				return
			default:
			}

			select {
			case subcount = <-this.childChannel:
				this.switchPhase(_EXECTIME)

				// current policy is to only count 'in' documents
				// from operators, not kv
				// add this.addInDocs(1) if this changes
				// this could be used for diagnostic purposes:
				// if docsIn != spans, somethimg has gone wrong
				// somewhere
				count += subcount
				n--
			case <-this.stopChannel:
				return
			}
		}

		av := value.NewAnnotatedValue(count)
		av.InheritCovers(parent)
		this.sendItem(av)
	})
}

func (this *IndexCountScan) scanCount(span *plan.Span, scanVector timestamp.Vector, context *Context) {
	dspan, empty, err := evalSpan(span, context)

	var count int64
	if err == nil && !empty {
		count, err = this.plan.Index().Count(dspan, context.ScanConsistency(), scanVector)
	}

	if err != nil {
		context.Error(errors.NewEvaluationError(err, "scanCount()"))
	}

	this.childChannel <- count
}

func (this *IndexCountScan) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
