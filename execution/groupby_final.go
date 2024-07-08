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
	"github.com/couchbase/query/value"
)

type FinalGroup struct {
	base
	groupBase
	plan *plan.FinalGroup
}

func NewFinalGroup(plan *plan.FinalGroup, context *Context) *FinalGroup {

	merge := func(v1 value.AnnotatedValue, v2 value.AnnotatedValue) value.AnnotatedValue {
		logging.Debugf("Invalid call to merge in FinalGroup")
		return v1
	}

	rv := &FinalGroup{
		plan: plan,
	}
	newBase(&rv.base, context)
	newGroupBase(&rv.groupBase, context, plan.CanSpill(), merge)
	rv.output = rv
	return rv
}

func (this *FinalGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFinalGroup(this)
}

func (this *FinalGroup) Copy() Operator {
	rv := &FinalGroup{
		plan: this.plan,
	}
	this.base.copy(&rv.base)
	this.groupBase.copy(&rv.groupBase)
	return rv
}

func (this *FinalGroup) PlanOp() plan.Operator {
	return this.plan
}

func (this *FinalGroup) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent, this.Release)
}

func (this *FinalGroup) processItem(item value.AnnotatedValue, context *Context) bool {
	// Generate the group key
	var gk string
	if len(this.plan.Keys()) > 0 {
		var e error
		gk, e = groupKey(item, this.plan.Keys(), &this.operatorCtx)
		if e != nil {
			context.Fatal(errors.NewEvaluationError(e, "GROUP key"))
			item.Recycle()
			return false
		}
	}

	// Get or seed the group value
	gv, set, err := this.groups.LoadOrStore(gk, item)
	if err != nil {
		context.Fatal(errors.NewEvaluationError(err, "GROUP key"))
		item.Recycle()
		return false
	} else if !set {
		context.Fatal(errors.NewDuplicateFinalGroupError())
		item.Recycle()
		return false
	}

	// Compute final aggregates
	aggregates := gv.GetAttachment(value.ATT_AGGREGATES)
	switch aggregates := aggregates.(type) {
	case map[string]value.Value:
		for _, agg := range this.plan.Aggregates() {
			v, e := agg.ComputeFinal(aggregates[agg.String()], &this.operatorCtx)
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
	if !this.stopped {
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
		} else if len(this.plan.Keys()) == 0 && groups_len == 0 && !this.stopped {
			// Mo matching inputs, so send default values
			av := value.NewAnnotatedValue(nil)
			aggregates := make(map[string]value.Value, len(this.plan.Aggregates()))
			av.SetAttachment(value.ATT_AGGREGATES, aggregates)
			for _, agg := range this.plan.Aggregates() {
				aggregates[agg.String()], _ = agg.Default(nil, &this.operatorCtx)
			}

			if context.UseRequestQuota() {
				if err := context.TrackValueSize(av.Size()); err != nil {
					context.Error(err)
					av.Recycle()
					this.Release()
					return
				}
			}
			this.sendItem(av)
		}
	}
	this.Release()
}

func (this *FinalGroup) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *FinalGroup) reopen(context *Context) bool {
	this.Release()
	return this.baseReopen(context)
}
