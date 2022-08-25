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
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

// Grouping of groups. Recursable.
type IntermediateGroup struct {
	base
	plan   *plan.IntermediateGroup
	groups *value.AnnotatedMap
}

func NewIntermediateGroup(plan *plan.IntermediateGroup, context *Context) *IntermediateGroup {

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
					context.Fatal(errors.NewMemoryQuotaExceededError())
				}
			}
		}
	}
	merge := func(v1 value.AnnotatedValue, v2 value.AnnotatedValue) value.AnnotatedValue {
		a1 := v1.GetAttachment("aggregates").(map[string]value.Value)
		a2 := v2.GetAttachment("aggregates").(map[string]value.Value)
		for _, agg := range plan.Aggregates() {
			a := agg.String()
			v, e := agg.CumulateIntermediate(a2[a], a1[a], nil)
			if e != nil {
				return nil
			}
			a1[a] = v
		}
		return v1
	}

	rv := &IntermediateGroup{
		plan:   plan,
		groups: value.NewAnnotatedMap(shouldSpill, trackMem, merge),
	}

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *IntermediateGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIntermediateGroup(this)
}

func (this *IntermediateGroup) Copy() Operator {
	rv := &IntermediateGroup{
		plan:   this.plan,
		groups: this.groups.Copy(),
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *IntermediateGroup) PlanOp() plan.Operator {
	return this.plan
}

func (this *IntermediateGroup) RunOnce(context *Context, parent value.Value) {
	defer this.groups.Release()
	this.runConsumer(this, context, parent)
}

func (this *IntermediateGroup) processItem(item value.AnnotatedValue, context *Context) bool {
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
	if gv == nil {

		// avoid recycling of seeding values
		this.groups.Set(gk, item)
		return true
	}
	// Cumulate aggregates
	part, ok := item.GetAttachment("aggregates").(map[string]value.Value)
	if !ok {
		context.Fatal(errors.NewInvalidValueError(
			fmt.Sprintf("Invalid partial aggregates %v of type %T", part, part)))
		item.Recycle()
		return false
	}

	if context.UseRequestQuota() {
		context.ReleaseValueSize(item.Size())
	}
	item.Recycle()

	cumulative := gv.GetAttachment("aggregates").(map[string]value.Value)
	if !ok {
		context.Fatal(errors.NewInvalidValueError(
			fmt.Sprintf("Invalid cumulative aggregates %v of type %T", cumulative, cumulative)))
		return false
	}

	for _, agg := range this.plan.Aggregates() {
		a := agg.String()
		v, e := agg.CumulateIntermediate(part[a], cumulative[a], context)
		if e != nil {
			context.Fatal(errors.NewGroupUpdateError(
				e, "Error updating intermediate GROUP value."))
			return false
		}

		cumulative[a] = v
	}

	return true
}

func (this *IntermediateGroup) afterItems(context *Context) {
	this.groups.Foreach(func(key string, av value.AnnotatedValue) bool {
		if !this.sendItem(av) {
			return false
		}
		return true
	})
	this.groups.Release()
}

func (this *IntermediateGroup) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *IntermediateGroup) reopen(context *Context) bool {
	this.groups.Release()
	return this.baseReopen(context)
}
