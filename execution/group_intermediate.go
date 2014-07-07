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

// Grouping of groups. Recursable.
type IntermediateGroup struct {
	base
	plan   *plan.IntermediateGroup
	groups map[string]value.AnnotatedValue
}

func NewIntermediateGroup(plan *plan.IntermediateGroup) *IntermediateGroup {
	rv := &IntermediateGroup{
		base:   newBase(),
		plan:   plan,
		groups: make(map[string]value.AnnotatedValue),
	}

	rv.output = rv
	return rv
}

func (this *IntermediateGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIntermediateGroup(this)
}

func (this *IntermediateGroup) Copy() Operator {
	return &IntermediateGroup{
		base:   this.base.copy(),
		plan:   this.plan,
		groups: make(map[string]value.AnnotatedValue),
	}
}

func (this *IntermediateGroup) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *IntermediateGroup) processItem(item value.AnnotatedValue, context *Context) bool {
	// Generate the group key
	var gk string
	if len(this.plan.Keys()) > 0 {
		var e error
		gk, e = groupKey(item, this.plan.Keys(), context)
		if e != nil {
			context.ErrorChannel() <- errors.NewError(e, "Error evaluating GROUP key.")
			return false
		}
	}

	// Get or seed the group value
	gv := this.groups[gk]
	if gv == nil {
		gv = item
		this.groups[gk] = gv
		return true
	}

	// Cumulate aggregates
	aggregates := gv.GetAttachment("aggregates")
	switch aggregates := aggregates.(type) {
	case map[algebra.Aggregate]value.Value:
		for agg, val := range aggregates {
			v, e := agg.CumulateIntermediate(item, val, context)
			if e != nil {
				context.ErrorChannel() <- errors.NewError(
					e, "Error updating GROUP value.")
				return false
			}
			aggregates[agg] = v
		}
		return true
	default:
		context.ErrorChannel() <- errors.NewError(nil, fmt.Sprintf(
			"Invalid or missing aggregates of type %T.", aggregates))
		return false
	}
}

func (this *IntermediateGroup) afterItems(context *Context) {
	for _, av := range this.groups {
		if !this.sendItem(av) {
			return
		}
	}
}
