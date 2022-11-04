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

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type Filter struct {
	base
	buildBitFilterBase
	docs     uint64
	plan     *plan.Filter
	aliasMap map[string]string
}

var _FILTER_OP_POOL util.FastPool

func init() {
	util.NewFastPool(&_FILTER_OP_POOL, func() interface{} {
		return &Filter{}
	})
}

func NewFilter(plan *plan.Filter, context *Context, aliasMap map[string]string) *Filter {
	rv := _FILTER_OP_POOL.Get().(*Filter)
	rv.plan = plan
	rv.aliasMap = aliasMap
	newBase(&rv.base, context)
	rv.execPhase = FILTER
	rv.output = rv
	return rv
}

func (this *Filter) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFilter(this)
}

func (this *Filter) Copy() Operator {
	rv := _FILTER_OP_POOL.Get().(*Filter)
	rv.plan = this.plan
	rv.aliasMap = this.aliasMap
	this.base.copy(&rv.base)
	return rv
}

func (this *Filter) PlanOp() plan.Operator {
	return this.plan
}

func (this *Filter) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Filter) beforeItems(context *Context, parent value.Value) bool {
	this.plan.Condition().EnableInlistHash(&this.operatorCtx)
	SetSearchInfo(this.aliasMap, parent, &this.operatorCtx, this.plan.Condition())
	buildBitFilters := this.plan.GetBuildBitFilters()
	if len(buildBitFilters) > 0 {
		this.createLocalBuildFilters(buildBitFilters)
	}
	return true
}

func (this *Filter) processItem(item value.AnnotatedValue, context *Context) bool {
	val, e := this.plan.Condition().Evaluate(item, &this.operatorCtx)
	if e != nil {
		context.Error(errors.NewEvaluationError(e, "filter"))
		return false
	}

	if val.Truth() {
		this.docs++
		if this.docs > _PHASE_UPDATE_COUNT {
			context.AddPhaseCount(FILTER, this.docs)
			this.docs = 0
		}
		if this.hasBuildBitFilter() && !this.buildBitFilters(item, &this.operatorCtx) {
			return false
		}
		return this.sendItem(item)
	} else {
		if context.UseRequestQuota() {
			context.ReleaseValueSize(item.Size())
		}
		item.Recycle()
		return true
	}
}

func (this *Filter) afterItems(context *Context) {
	this.plan.Condition().ResetMemory(&this.operatorCtx)
	if this.docs > 0 {
		context.AddPhaseCount(FILTER, this.docs)
		this.docs = 0
	}
	if this.hasBuildBitFilter() {
		this.setBuildBitFilters(this.plan.Alias(), context)
	}
}

func (this *Filter) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *Filter) Done() {
	this.baseDone()
	if this.isComplete() {
		this.docs = 0
		this.aliasMap = nil
		this.plan = nil
		_FILTER_OP_POOL.Put(this)
	}
}
