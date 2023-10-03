//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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
	rv.mk.validate = plan.Term().ValidateKeys()
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
	this.mk.validate = this.mk.validate
	return rv
}

func (this *Nest) PlanOp() plan.Operator {
	return this.plan
}

func (this *Nest) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent, nil)
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

func (this *Nest) beforeItems(context *Context, item value.Value) bool {
	this.mk.reset()
	return true
}

func (this *Nest) afterItems(context *Context) {
	this.flushBatch(context)
	this.mk.report(context, this.plan.Keyspace().Name)
}

func (this *Nest) flushBatch(context *Context) bool {
	defer this.releaseBatch(context)

	if len(this.joinBatch) == 0 || !this.isRunning() {
		return true
	}

	keyCount := _STRING_KEYCOUNT_POOL.Get()
	pairMap := _STRING_ANNOTATED_POOL.Get()

	defer _STRING_KEYCOUNT_POOL.Put(keyCount)
	defer _STRING_ANNOTATED_POOL.Put(pairMap)

	fetchOk := this.joinFetch(this.plan.Keyspace(), this.plan.SubPaths(), nil, keyCount, pairMap, context)

	this.validateKeys(pairMap)

	return fetchOk && this.nestEntries(keyCount, pairMap, this.plan.Outer(), this.plan.OnFilter(), this.plan.Term().Alias(), &this.operatorCtx)
}

func (this *Nest) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
