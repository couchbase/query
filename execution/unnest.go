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
	"github.com/couchbase/query/value"
)

type Unnest struct {
	base
	buildBitFilterBase
	plan *plan.Unnest
}

func NewUnnest(plan *plan.Unnest, context *Context) *Unnest {
	rv := &Unnest{
		plan: plan,
	}

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *Unnest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnnest(this)
}

func (this *Unnest) Copy() Operator {
	rv := &Unnest{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *Unnest) PlanOp() plan.Operator {
	return this.plan
}

func (this *Unnest) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Unnest) beforeItems(context *Context, parent value.Value) bool {
	filter := this.plan.Filter()
	if filter != nil {
		filter.EnableInlistHash(&this.operatorCtx)
	}
	buildBitFilters := this.plan.GetBuildBitFilters()
	if len(buildBitFilters) > 0 {
		this.createLocalBuildFilters(buildBitFilters)
	}
	return true
}

func (this *Unnest) processItem(item value.AnnotatedValue, context *Context) bool {
	ev, err := this.plan.Term().Expression().Evaluate(item, &this.operatorCtx)
	if err != nil {
		context.Error(errors.NewEvaluationError(err, "UNNEST path"))
		return false
	}

	// not an array, treat as outer unnest
	if ev.Type() != value.ARRAY {
		return !this.plan.Term().Outer() || this.sendItem(item)
	}

	idx := 0

	// empty, treat as outer unnest
	act, ok := ev.Index(idx)
	if act.Type() == value.MISSING && !ok {
		return !this.plan.Term().Outer() || this.sendItem(item)
	}

	filter := this.plan.Filter()
	buildBitFltr := this.hasBuildBitFilter()

	// Attach and send
	var av value.AnnotatedValue
	var actv value.AnnotatedValue
	for {
		if actv == nil {
			actv = value.NewAnnotatedValue(act)
		} else {
			actv.SetValue(act)
		}
		actv.SetAttachment("unnest_position", idx)

		idx++
		nextAct, isValidIndex := ev.Index(idx)

		if !isValidIndex {
			if av != nil {
				av.Recycle()
			}
			av = item
			av.SetField(this.plan.Alias(), actv)
		} else {
			if av == nil {
				av = value.NewAnnotatedValue(item.Copy())
				av.SetField(this.plan.Alias(), actv)
			}
		}

		pass := true
		if filter != nil {
			result, err := filter.Evaluate(av, &this.operatorCtx)
			if err != nil {
				context.Error(errors.NewEvaluationError(err, "unnest filter"))
				return false
			}
			pass = result.Truth()
			result = nil
		}

		if pass {
			if buildBitFltr && !this.buildBitFilters(av, &this.operatorCtx) {
				av.Recycle()
				return false
			}

			if context.UseRequestQuota() {
				sz := av.Size()
				if !isValidIndex {
					// for the last item, only track the growth
					sz = actv.Size() + uint64(len(this.plan.Alias()))
				}
				err := context.TrackValueSize(sz)
				if err != nil {
					context.Error(err)
					av.Recycle()
					return false
				}
			}
			if !this.sendItem(av) {
				av.Recycle()
				return false
			} else {
				av = nil
				actv = nil
			}
		}

		// no more
		if !isValidIndex {
			if actv != nil {
				actv.Recycle()
			}
			if av != nil {
				av.Recycle()
			}
			break
		}
		act = nextAct
	}

	return true
}

func (this *Unnest) afterItems(context *Context) {
	filter := this.plan.Filter()
	if filter != nil {
		filter.ResetMemory(&this.operatorCtx)
	}
	if this.hasBuildBitFilter() {
		this.setBuildBitFilters(this.plan.Alias(), context)
	}
}

func (this *Unnest) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

var _EMPTY_ACTUALS []interface{}
