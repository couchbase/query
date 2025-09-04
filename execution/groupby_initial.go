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

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
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
	this.runConsumer(this, context, parent, this.Release)
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

	recycle := false
	releaseSize := uint64(0)

	// Get or seed the group value
	gv := this.groups[gk]
	if gv == nil {
		gv = item
		this.groups[gk] = gv
		aggregates := make(map[string]value.Value, len(this.plan.Aggregates()))
		gv.SetAttachment(value.ATT_AGGREGATES, aggregates)
		for _, agg := range this.plan.Aggregates() {
			aggregates[agg.String()], _ = agg.Default(nil, &this.operatorCtx)
		}
	} else {
		if context.UseRequestQuota() {
			releaseSize = item.Size()
		}
		recycle = true
	}

	// Cumulate aggregates
	aggregates, ok := gv.GetAttachment(value.ATT_AGGREGATES).(map[string]value.Value)
	if !ok {
		context.Fatal(errors.NewInvalidValueError(fmt.Sprintf("Invalid aggregates %v of type %T", aggregates, aggregates)))
		item.Recycle()
		return false
	}

	for _, agg := range this.plan.Aggregates() {
		// WARNING: do not cache agg.String() - it may change during CumulateInitial
		a := agg.String()
		pv := aggregates[a]
		if pv == nil {
			// Log an error and explicitly panic
			// If we attempt to recover from this situation here we'll probably be producing inaccurate results - better to halt
			logging.Severef("Aggregate '%s' not found for InitialGroup in aggregates (%v) for group key '%v'", a, aggregates, gk)
			panic("Aggregate not found")
		}
		v, e := agg.CumulateInitial(item, pv, &this.operatorCtx)
		if e != nil {
			context.Fatal(errors.NewGroupUpdateError(e, "Error updating initial GROUP value."))
			item.Recycle()
			return false
		}

		/* MB-65862. In group intermidiate, final uses v.Equals(pv) == value.FALSE_VALUE.
		   Because it only recycle. Here we need track even if v, pv contains NULL. Comparison
		   returns value.VALUE_NULL and we need to track the value.
		*/
		if v.Equals(pv) != value.TRUE_VALUE {
			// MB-65246, MB-68296, for ARRAY_AGG() Track happens in the aggregate itself
			if array_agg, ok := agg.(*algebra.ArrayAgg); ok {
				if array_agg.Distinct() {
					recycle = false
					releaseSize = 0
				}
			} else {
				// maintain a reference count for each aggregate as appropriate
				v.Track()
				pv.Recycle()
			}
		}
		b := agg.String()
		aggregates[b] = v
		// delete the previous key if agg.String() has changed
		if a != b {
			delete(aggregates, a)
		}
	}

	// Update the Group Key's entry in the Map with the Group As field in the item
	if this.plan.GroupAs() != "" {
		groupAsAlias := this.plan.GroupAs()
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
		groupAsField, ok := gv.Field(groupAsAlias)
		if !ok {
			groupAsField = value.NewTrackedValue(make([]interface{}, 0, len(groupAs)))
		}

		groupAsVal := value.NewTrackedValue(groupAs)
		groupAsField, ok = groupAsField.Append([]interface{}{groupAsVal})
		if !ok {
			context.Fatal(errors.NewGroupUpdateError(nil, fmt.Sprintf("Group As append failed for alias %s", groupAsAlias)))
			item.Recycle()
			return false
		}

		gv.SetField(groupAsAlias, groupAsField)
		if releaseSize > 0 {
			// don't release the quota associated with the item since it has been included in the payload
			if releaseSize > groupAsVal.Size() {
				releaseSize -= groupAsVal.Size()
			} else {
				releaseSize = 0
			}
		}
	}

	if releaseSize > 0 {
		context.ReleaseValueSize(releaseSize)
	}
	if recycle {
		item.Recycle()
	}

	return true
}

func (this *InitialGroup) afterItems(context *Context) {
	if !this.stopped {
		for _, av := range this.groups {
			if !this.sendItem(av) {
				return
			}
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
	this.Release()
	rv := this.baseReopen(context)
	if rv {
		this.groups = make(map[string]value.AnnotatedValue)
	}
	return rv
}

func (this *InitialGroup) Release() {
	if this.groups != nil {
		for k, _ := range this.groups {
			delete(this.groups, k)
		}
		this.groups = nil
	}
}
