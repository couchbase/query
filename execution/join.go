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

type Join struct {
	joinBase
	plan *plan.Join
}

func NewJoin(plan *plan.Join, context *Context) *Join {
	rv := &Join{
		plan: plan,
	}

	newJoinBase(&rv.joinBase, context)
	rv.execPhase = JOIN
	rv.output = rv
	return rv
}

func (this *Join) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitJoin(this)
}

func (this *Join) Copy() Operator {
	rv := &Join{
		plan: this.plan,
	}
	this.joinBase.copy(&rv.joinBase)
	return rv
}

func (this *Join) PlanOp() plan.Operator {
	return this.plan
}

func (this *Join) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent, nil)
}

func (this *Join) processItem(item value.AnnotatedValue, context *Context) bool {
	keys, ok := this.evaluateKey(this.plan.Term().JoinKeys(), item, context)
	if !ok {
		if context.UseRequestQuota() {
			context.ReleaseValueSize(item.Size())
		}
		item.Recycle()
		return false
	}

	doc := value.AnnotatedJoinPair{Value: item, Keys: keys}
	return this.joinEnbatch(doc, this, context)
}

func (this *Join) afterItems(context *Context) {
	this.flushBatch(context)
	this.releaseBatch(context)
}

func (this *Join) flushBatch(context *Context) bool {
	defer this.resetBatch(context)

	if len(this.joinBatch) == 0 {
		return true
	}

	keyCount := _STRING_KEYCOUNT_POOL.Get()
	pairMap := _STRING_ANNOTATED_POOL.Get()

	defer _STRING_KEYCOUNT_POOL.Put(keyCount)
	defer _STRING_ANNOTATED_POOL.Put(pairMap)

	fetchOk := this.joinFetch(this.plan.Keyspace(), keyCount, pairMap, context)

	return fetchOk && this.joinEntries(keyCount, pairMap, this.plan.Outer(), this.plan.OnFilter(), this.plan.Term().Alias(), context)
}

func (this *Join) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
