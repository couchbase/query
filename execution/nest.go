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

	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type Nest struct {
	joinBase
	plan *plan.Nest
}

func NewNest(plan *plan.Nest, context *Context) *Nest {
	rv := &Nest{
		plan: plan,
	}

	newJoinBase(&rv.joinBase, context)
	rv.execPhase = NEST
	rv.output = rv
	return rv
}

func (this *Nest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitNest(this)
}

func (this *Nest) Copy() Operator {
	rv := &Nest{
		plan: this.plan,
	}
	this.joinBase.copy(&rv.joinBase)
	return rv
}

func (this *Nest) PlanOp() plan.Operator {
	return this.plan
}

func (this *Nest) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Nest) processItem(item value.AnnotatedValue, context *Context) bool {
	keys, ok := this.evaluateKey(this.plan.Term().JoinKeys(), item, context)
	if !ok {
		if context.UseRequestQuota() {
			context.ReleaseValueSize(item.Size())
		}
		return false
	}

	doc := value.AnnotatedJoinPair{Value: item, Keys: keys}
	return this.joinEnbatch(doc, this, context)
}

func (this *Nest) afterItems(context *Context) {
	this.flushBatch(context)
}

func (this *Nest) flushBatch(context *Context) bool {
	defer this.releaseBatch(context)

	if len(this.joinBatch) == 0 {
		return true
	}

	keyCount := _STRING_KEYCOUNT_POOL.Get()
	pairMap := _STRING_ANNOTATED_POOL.Get()

	defer _STRING_KEYCOUNT_POOL.Put(keyCount)
	defer _STRING_ANNOTATED_POOL.Put(pairMap)

	fetchOk := this.joinFetch(this.plan.Keyspace(), keyCount, pairMap, context)

	return fetchOk && this.nestEntries(keyCount, pairMap, this.plan.Outer(), this.plan.OnFilter(), this.plan.Term().Alias(), context)
}

func (this *Nest) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
