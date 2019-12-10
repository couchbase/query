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
			return false
		}
	}

	// Get or seed the group value
	gv := this.groups[gk]
	if gv == nil {
		gv = item
		this.groups[gk] = gv

		aggregates := make(map[string]value.Value, len(this.plan.Aggregates()))
		gv.SetAttachment("aggregates", aggregates)
		for _, agg := range this.plan.Aggregates() {
			aggregates[agg.String()], _ = agg.Default(nil, context)
		}
	}

	// Cumulate aggregates
	aggregates, ok := gv.GetAttachment("aggregates").(map[string]value.Value)
	if !ok {
		context.Fatal(errors.NewInvalidValueError(
			fmt.Sprintf("Invalid aggregates %v of type %T", aggregates, aggregates)))
		return false
	}

	for _, agg := range this.plan.Aggregates() {
		v, e := agg.CumulateInitial(item, aggregates[agg.String()], context)
		if e != nil {
			context.Fatal(errors.NewGroupUpdateError(e, "Error updating initial GROUP value."))
			return false
		}

		aggregates[agg.String()] = v
	}

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
