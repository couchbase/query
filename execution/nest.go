//  Copyright (c) 2014 Couchbase, Inc.
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

	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type Nest struct {
	joinBase
	plan *plan.Nest
}

func NewNest(plan *plan.Nest) *Nest {
	rv := &Nest{
		joinBase: newJoinBase(),
		plan:     plan,
	}

	rv.output = rv
	return rv
}

func (this *Nest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitNest(this)
}

func (this *Nest) Copy() Operator {
	return &Nest{
		joinBase: this.joinBase.copy(),
		plan:     this.plan,
	}
}

func (this *Nest) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
	t := this.duration - this.chanTime
	context.AddPhaseTime(NEST, t)
	this.addTime(t)
}

func (this *Nest) processItem(item value.AnnotatedValue, context *Context) bool {
	keys, ok := this.evaluateKey(this.plan.Term().Keys(), item, context)
	if !ok {
		return false
	}

	doc := value.AnnotatedJoinPair{Value: item, Keys: keys}
	return this.joinEnbatch(doc, this, context)
}

func (this *Nest) afterItems(context *Context) {
	this.flushBatch(context)
}

func (this *Nest) flushBatch(context *Context) bool {
	defer this.releaseBatch()

	if len(this.joinBatch) == 0 {
		return true
	}

	timer := time.Now()

	keyCount := _STRING_KEYCOUNT_POOL.Get()
	pairMap := _STRING_ANNOTATED_POOL.Get()

	defer _STRING_KEYCOUNT_POOL.Put(keyCount)
	defer _STRING_ANNOTATED_POOL.Put(pairMap)
	defer func() {
		this.duration += time.Since(timer)
	}()

	fetchOk := this.joinFetch(this.plan.Keyspace(), keyCount, pairMap, context)

	return fetchOk && this.nestEntries(keyCount, pairMap, this.plan.Outer(), this.plan.Term().Alias())
}

func (this *Nest) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *Nest) Done() {
}
