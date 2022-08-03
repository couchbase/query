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
	"github.com/couchbase/query/value"
)

// Grouping of input data.
type InitialGroup struct {
	base
	plan   *plan.InitialGroup
	groups map[string]value.AnnotatedValue
}

func NewInitialGroup(plan *plan.InitialGroup, context *Context) *InitialGroup {
	rv := &InitialGroup{
		plan:   plan,
		groups: make(map[string]value.AnnotatedValue),
	}

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *InitialGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitInitialGroup(this)
}

func (this *InitialGroup) Copy() Operator {
	rv := &InitialGroup{
		plan:   this.plan,
		groups: make(map[string]value.AnnotatedValue),
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *InitialGroup) PlanOp() plan.Operator {
	return this.plan
}

func (this *InitialGroup) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *InitialGroup) processItem(item value.AnnotatedValue, context *Context) bool {
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
	gv := this.groups[gk]
	handleQuota := false
	if gv == nil {

		// avoid recycling of seeding values
		item.Track()
		gv = item
		this.groups[gk] = gv

		aggregates := make(map[string]value.Value, len(this.plan.Aggregates()))
		gv.SetAttachment("aggregates", aggregates)
		for _, agg := range this.plan.Aggregates() {
			aggregates[agg.String()], _ = agg.Default(nil, context)
		}
	} else {
		handleQuota = context.UseRequestQuota()
	}

	// Cumulate aggregates
	aggregates, ok := gv.GetAttachment("aggregates").(map[string]value.Value)
	if !ok {
		context.Fatal(errors.NewInvalidValueError(
			fmt.Sprintf("Invalid aggregates %v of type %T", aggregates, aggregates)))
		item.Recycle()
		return false
	}

	for _, agg := range this.plan.Aggregates() {
		v, e := agg.CumulateInitial(item, aggregates[agg.String()], context)
		if e != nil {
			context.Fatal(errors.NewGroupUpdateError(e, "Error updating initial GROUP value."))
			item.Recycle()
			return false
		}

		aggregates[agg.String()] = v
	}
	if handleQuota {
		context.ReleaseValueSize(item.Size())
	}
	item.Recycle()

	return true
}

func (this *InitialGroup) afterItems(context *Context) {
	for _, av := range this.groups {
		if !this.sendItem(av) {
			return
		}
	}
}

func (this *InitialGroup) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *InitialGroup) reopen(context *Context) bool {
	rv := this.baseReopen(context)
	this.groups = make(map[string]value.AnnotatedValue)
	return rv
}
