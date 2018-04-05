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

type Fetch struct {
	base
	plan       *plan.Fetch
	batchSize  int
	fetchCount uint64
}

func NewFetch(plan *plan.Fetch, context *Context) *Fetch {
	rv := &Fetch{
		plan:      plan,
		batchSize: context.GetPipelineBatch(),
	}

	newBase(&rv.base, context)
	rv.execPhase = FETCH
	rv.output = rv
	return rv
}

func (this *Fetch) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFetch(this)
}

func (this *Fetch) Copy() Operator {
	rv := &Fetch{plan: this.plan, batchSize: this.batchSize}
	this.base.copy(&rv.base)
	return rv
}

func (this *Fetch) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Fetch) processItem(item value.AnnotatedValue, context *Context) bool {
	ok := this.enbatchSize(item, this, this.batchSize, context)
	if ok {
		this.fetchCount++
		if this.fetchCount >= uint64(this.batchSize) {
			context.AddPhaseCount(FETCH, this.fetchCount)
			this.fetchCount = 0
		}
	}
	return ok
}

func (this *Fetch) afterItems(context *Context) {
	this.flushBatch(context)
	context.SetSortCount(0)
	context.AddPhaseCount(FETCH, this.fetchCount)
}

func (this *Fetch) flushBatch(context *Context) bool {
	defer this.releaseBatch(context)
	if this.batchSize < int(context.PipelineCap()) {
		defer func() {
			this.batchSize = int(context.PipelineCap())
		}()
	}

	if len(this.batch) == 0 {
		return true
	}

	fetchKeys := _STRING_POOL.Get()
	defer _STRING_POOL.Put(fetchKeys)

	keyCount := _STRING_KEYCOUNT_POOL.Get()
	defer _STRING_KEYCOUNT_POOL.Put(keyCount)

	fetchMap := _STRING_ANNOTATED_POOL.Get()
	defer _STRING_ANNOTATED_POOL.Put(fetchMap)

	for _, av := range this.batch {
		key, ok := this.requireKey(av, context)
		if !ok {
			return false
		}

		v, ok := keyCount[key]
		if !ok {
			fetchKeys = append(fetchKeys, key)
			v = 0
		}
		keyCount[key] = v + 1
	}

	this.switchPhase(_SERVTIME)

	// Fetch
	errs := this.plan.Keyspace().Fetch(fetchKeys, fetchMap, context, this.plan.SubPaths())

	this.switchPhase(_EXECTIME)

	fetchOk := true
	for _, err := range errs {
		context.Error(err)
		if err.IsFatal() {
			fetchOk = false
		}
	}

	// Preserve order of keys
	for _, av := range this.batch {
		key, ok := this.requireKey(av, context)
		if !ok {
			return false
		}

		fv := fetchMap[key]
		if fv != nil {
			if keyCount[key] > 1 {
				fv = value.NewAnnotatedValue(fv.Copy())
				keyCount[key]--
			}

			av.SetField(this.plan.Term().Alias(), fv)

			if !this.sendItem(av) {
				return false
			}
		}
	}

	return fetchOk
}

func (this *Fetch) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

type DummyFetch struct {
	base
	plan *plan.DummyFetch
}

func NewDummyFetch(plan *plan.DummyFetch, context *Context) *DummyFetch {
	rv := &DummyFetch{
		plan: plan,
	}

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *DummyFetch) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDummyFetch(this)
}

func (this *DummyFetch) Copy() Operator {
	rv := &DummyFetch{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *DummyFetch) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *DummyFetch) processItem(item value.AnnotatedValue, context *Context) bool {
	item.SetField(this.plan.Term().Alias(), item.Copy())
	return this.sendItem(item)
}

func (this *DummyFetch) afterItems(context *Context) {
	context.SetSortCount(0)
}

func (this *DummyFetch) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
