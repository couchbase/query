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
	"fmt"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type FinalGroup struct {
	base
	plan   *plan.FinalGroup
	groups *value.AnnotatedMap
}

func NewFinalGroup(plan *plan.FinalGroup, context *Context) *FinalGroup {

	var shouldSpill func(uint64, uint64) bool
	if context.UseRequestQuota() {
		shouldSpill = func(c uint64, n uint64) bool {
			return (c+n) > context.ProducerThrottleQuota() && context.CurrentQuotaUsage() > _GROUP_QUOTA_THRESHOLD
		}
	} else {
		maxSize := context.AvailableMemory()
		if maxSize > 0 {
			maxSize = uint64(float64(maxSize) / float64(util.NumCPU()) * _GROUP_AVAILABLE_MEMORY_THRESHOLD)
		}
		if maxSize < _MIN_SIZE {
			maxSize = _MIN_SIZE
		}
		shouldSpill = func(c uint64, n uint64) bool {
			return (c + n) > maxSize
		}
	}

	trackMem := func(size int64) {
		if context.UseRequestQuota() {
			if size < 0 {
				context.ReleaseValueSize(uint64(-size))
			} else {
				if err := context.TrackValueSize(uint64(size)); err != nil {
					context.Fatal(err)
				}
			}
		}
	}
	merge := func(v1 value.AnnotatedValue, v2 value.AnnotatedValue) value.AnnotatedValue {
		logging.Debugf("Invalid call to merge in FinalGroup")
		return v1
	}

	rv := &FinalGroup{
		plan:   plan,
		groups: value.NewAnnotatedMap(shouldSpill, trackMem, merge),
	}

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *FinalGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFinalGroup(this)
}

func (this *FinalGroup) Copy() Operator {
	rv := &FinalGroup{
		plan:   this.plan,
		groups: this.groups.Copy(),
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *FinalGroup) PlanOp() plan.Operator {
	return this.plan
}

func (this *FinalGroup) RunOnce(context *Context, parent value.Value) {
	defer this.groups.Release()
	this.runConsumer(this, context, parent)
}

func (this *FinalGroup) processItem(item value.AnnotatedValue, context *Context) bool {
	// Generate the group key
	var gk string
	if len(this.plan.Keys()) > 0 {
		var e error
		gk, e = groupKey(item, this.plan.Keys(), context)
		if e != nil {
			context.Fatal(errors.NewEvaluationError(e, "GROUP key"))
			item.Recycle()
			return false
		}
	}

	// Get or seed the group value
	gv := this.groups.Get(gk)
	if gv != nil {
		context.Fatal(errors.NewDuplicateFinalGroupError())
		item.Recycle()
		return false
	}

	gv = item
	err := this.groups.Set(gk, gv)
	if err != nil {
		context.Fatal(err)
		item.Recycle()
		return false
	}

	// Compute final aggregates
	aggregates := gv.GetAttachment("aggregates")
	switch aggregates := aggregates.(type) {
	case map[string]value.Value:
		for _, agg := range this.plan.Aggregates() {
			v, e := agg.ComputeFinal(aggregates[agg.String()], context)
			if e != nil {
				context.Fatal(errors.NewGroupUpdateError(e, "Error updating final GROUP value."))
				item.Recycle()
				return false
			}

			aggregates[agg.String()] = v
		}
		return true
	default:
		context.Fatal(errors.NewInvalidValueError(fmt.Sprintf("Invalid or missing aggregates of type %T.", aggregates)))
		item.Recycle()
		return false
	}
}

func (this *FinalGroup) afterItems(context *Context) {
	groups_len := 0
	err := this.groups.Foreach(func(key string, av value.AnnotatedValue) bool {
		groups_len++
		if !this.sendItem(av) {
			return false
		}
		return true
	})
	if err != nil {
		context.Error(err)
		return
	}

	// Mo matching inputs, so send default values
	if len(this.plan.Keys()) == 0 && groups_len == 0 {
		av := value.NewAnnotatedValue(nil)
		aggregates := make(map[string]value.Value, len(this.plan.Aggregates()))
		av.SetAttachment("aggregates", aggregates)
		for _, agg := range this.plan.Aggregates() {
			aggregates[agg.String()], _ = agg.Default(nil, context)
		}

		if context.UseRequestQuota() {
			if err := context.TrackValueSize(av.Size()); err != nil {
				context.Error(err)
				av.Recycle()
			}
		} else {
			this.sendItem(av)
		}
	}
}

func (this *FinalGroup) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *FinalGroup) reopen(context *Context) bool {
	this.groups.Release()
	return this.baseReopen(context)
}
