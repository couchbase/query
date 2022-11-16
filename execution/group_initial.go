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

const (
	_GROUP_QUOTA_THRESHOLD            = 0.75 // % quota usage
	_GROUP_AVAILABLE_MEMORY_THRESHOLD = 0.15 // % per CPU free memory
)

// Grouping of input data.
type InitialGroup struct {
	base
	plan   *plan.InitialGroup
	groups *value.AnnotatedMap
}

func NewInitialGroup(plan *plan.InitialGroup, context *Context) *InitialGroup {

	var shouldSpill func(uint64, uint64) bool
	if plan.CanSpill() && context.UseRequestQuota() {
		shouldSpill = func(c uint64, n uint64) bool {
			return (c+n) > context.ProducerThrottleQuota() && context.CurrentQuotaUsage() > _GROUP_QUOTA_THRESHOLD
		}
	} else if plan.CanSpill() {
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

		// If the Group As clause is present, merge the values of the Group As field in the entries as well
		if plan.GroupAs() != "" {
			groupAsv1, _ := v1.Field(plan.GroupAs())
			groupAsv2, _ := v2.Field(plan.GroupAs())

			act1, _ := groupAsv1.Actual().([]interface{})
			act2, _ := groupAsv2.Actual().([]interface{})
			act := append(act1, act2...)
			v1.SetField(plan.GroupAs(), act)
		}
		return v1
	}

	rv := &InitialGroup{
		plan:   plan,
		groups: value.NewAnnotatedMap(shouldSpill, trackMem, merge),
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
		groups: this.groups.Copy(),
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *InitialGroup) PlanOp() plan.Operator {
	return this.plan
}

func (this *InitialGroup) RunOnce(context *Context, parent value.Value) {
	defer this.groups.Release()
	this.runConsumer(this, context, parent)
}

func (this *InitialGroup) processItem(item value.AnnotatedValue, context *Context) bool {
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
	recycle := false
	handleQuota := false

	gv, set, err := this.groups.LoadOrStore(gk, item)

	if err != nil {
		context.Fatal(errors.NewEvaluationError(err, "GROUP key"))
		item.Recycle()
		return false
	} else if set {
		aggregates := make(map[string]value.Value, len(this.plan.Aggregates()))
		gv.SetAttachment("aggregates", aggregates)
		for _, agg := range this.plan.Aggregates() {
			aggregates[agg.String()], _ = agg.Default(nil, &this.operatorCtx)
		}
	} else {
		handleQuota = context.UseRequestQuota()
		recycle = true
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
		v, e := agg.CumulateInitial(item, aggregates[agg.String()], &this.operatorCtx)
		if e != nil {
			context.Fatal(errors.NewGroupUpdateError(e, "Error updating initial GROUP value."))
			item.Recycle()
			return false
		}

		aggregates[agg.String()] = v
	}

	// Update the Group Key's entry in the Map with the Group As field in the item
	if this.plan.GroupAs() != "" {
		// Create the entry for the Group As array field
		groupAs := make(map[string]interface{}, len(this.plan.GroupAsFields()))
		itemAct := item.Actual().(map[string]interface{})

		// Only add the allowed group as fields from the item to the Group As entry
		for _, k := range this.plan.GroupAsFields() {
			field, ok := itemAct[k]
			if ok {
				groupAs[k] = field
			}
		}

		// Add the Group As field to the groupKey's entry in the map
		groupAsField, ok := gv.Field(this.plan.GroupAs())

		var act []interface{}
		if !ok {
			act = make([]interface{}, 0, 1)
		} else {
			act = groupAsField.Actual().([]interface{})
		}

		groupAsVal := value.NewValue(groupAs)
		act = append(act, groupAsVal)
		gv.SetField(this.plan.GroupAs(), value.NewValue(act))

		err := this.groups.AdjustSize(groupAsVal.Size()) // account for added field
		if err != nil {
			context.Fatal(err)
			return false
		}

	}

	if handleQuota {
		context.ReleaseValueSize(item.Size())
	}
	if recycle {
		item.Recycle()
	}

	return true
}

func (this *InitialGroup) afterItems(context *Context) {
	err := this.groups.Foreach(func(key string, av value.AnnotatedValue) bool {
		if !this.sendItem(av) {
			return false
		}
		return true
	})
	if err != nil {
		context.Error(err)
	}
	this.groups.Release()
}

func (this *InitialGroup) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *InitialGroup) reopen(context *Context) bool {
	this.groups.Release()
	return this.baseReopen(context)
}
