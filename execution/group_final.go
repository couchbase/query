//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execution

import (
	"fmt"

	"github.com/couchbaselabs/query/algebra"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

// Compute DistinctCount() and Avg().
type FinalGroup struct {
	base
	plan   *plan.FinalGroup
	groups map[string]value.AnnotatedValue
}

func NewFinalGroup(plan *plan.FinalGroup) *FinalGroup {
	rv := &FinalGroup{
		base:   newBase(),
		plan:   plan,
		groups: make(map[string]value.AnnotatedValue),
	}

	rv.output = rv
	return rv
}

func (this *FinalGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFinalGroup(this)
}

func (this *FinalGroup) Copy() Operator {
	return &FinalGroup{
		base:   this.base.copy(),
		plan:   this.plan,
		groups: make(map[string]value.AnnotatedValue),
	}
}

func (this *FinalGroup) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *FinalGroup) processItem(item value.AnnotatedValue, context *Context) bool {
	// Generate the group key
	var gk string
	if len(this.plan.Keys()) > 0 {
		var e error
		gk, e = groupKey(item, this.plan.Keys(), context)
		if e != nil {
			context.Error(errors.NewError(e, "Error evaluating GROUP key."))
			return false
		}
	}

	// Get or seed the group value
	gv := this.groups[gk]
	if gv != nil {
		context.Error(errors.NewError(nil, "Duplicate final GROUP."))
		return false
	}

	gv = item
	this.groups[gk] = gv

	// Compute final aggregates
	aggregates := gv.GetAttachment("aggregates")
	switch aggregates := aggregates.(type) {
	case map[algebra.Aggregate]value.Value:
		for agg, val := range aggregates {
			v, e := agg.ComputeFinal(val, context)
			if e != nil {
				context.Error(errors.NewError(
					e, "Error updating GROUP value."))
				return false
			}
			aggregates[agg] = v
		}
		return true
	default:
		context.Error(errors.NewError(nil, fmt.Sprintf(
			"Invalid or missing aggregates of type %T.", aggregates)))
		return false
	}
}

func (this *FinalGroup) afterItems(context *Context) {
	if len(this.groups) > 0 {
		for _, av := range this.groups {
			if !this.sendItem(av) {
				return
			}
		}
	} else {
		av := value.NewAnnotatedValue(nil)
		aggregates := make(map[algebra.Aggregate]value.Value, len(this.plan.Aggregates()))
		av.SetAttachment("aggregates", aggregates)

		for _, agg := range this.plan.Aggregates() {
			aggregates[agg] = agg.Default()
		}

		this.sendItem(av)
	}
}
