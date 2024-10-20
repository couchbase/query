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

type Alias struct {
	base
	buildBitFilterBase
	plan      *plan.Alias
	parentVal value.Value
}

func NewAlias(plan *plan.Alias, context *Context) *Alias {
	rv := &Alias{
		plan: plan,
	}

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *Alias) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAlias(this)
}

func (this *Alias) Copy() Operator {
	rv := &Alias{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *Alias) PlanOp() plan.Operator {
	return this.plan
}

func (this *Alias) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent, nil)
}

func (this *Alias) beforeItems(context *Context, parent value.Value) bool {
	this.parentVal = parent
	buildBitFilters := this.plan.GetBuildBitFilters()
	if len(buildBitFilters) > 0 {
		this.createLocalBuildFilters(buildBitFilters)
	}
	return true
}

func (this *Alias) processItem(item value.AnnotatedValue, context *Context) bool {
	cv := value.NewNestedScopeValue(this.parentVal)
	av := value.NewAnnotatedValue(cv)
	av.ShareAnnotations(item)
	av.SetField(this.plan.Alias(), item)
	item.Recycle() // tracked as a field now
	if this.hasBuildBitFilter() && !this.buildBitFilters(av, &this.operatorCtx) {
		av.Recycle()
		return false
	}

	if context.UseRequestQuota() {
		err := context.TrackValueSize(av.Size() - item.Size())
		if err != nil {
			context.Error(err)
			av.Recycle()
			return false
		}
	}
	return this.sendItem(av)
}

func (this *Alias) afterItems(context *Context) {
	if this.hasBuildBitFilter() {
		this.setBuildBitFilters(this.plan.Alias(), context)
	}
}

func (this *Alias) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *Alias) reopen(context *Context) bool {
	rv := this.baseReopen(context)
	this.parentVal = nil
	return rv
}

func (this *Alias) Done() {
	this.baseDone()
	this.parentVal = nil
}
