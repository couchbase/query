//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execute

import (
	_ "fmt"

	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

// Grouping of input data.
type InitialGroup struct {
	base
	plan *plan.InitialGroup
}

// Grouping of groups. Recursable.
type IntermediateGroup struct {
	base
	plan *plan.IntermediateGroup
}

// Compute DistinctCount() and Avg().
type FinalGroup struct {
	base
	plan *plan.FinalGroup
}

func NewInitialGroup(plan *plan.InitialGroup) *InitialGroup {
	rv := &InitialGroup{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *InitialGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitInitialGroup(this)
}

func (this *InitialGroup) Copy() Operator {
	return &InitialGroup{this.base.copy(), this.plan}
}

func (this *InitialGroup) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *InitialGroup) beforeItems(context *Context, parent value.Value) bool {
	return true
}

func (this *InitialGroup) processItem(item value.AnnotatedValue, context *Context) bool {
	return true
}

func (this *InitialGroup) afterItems(context *Context) {
}

func NewIntermediateGroup(plan *plan.IntermediateGroup) *IntermediateGroup {
	rv := &IntermediateGroup{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *IntermediateGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIntermediateGroup(this)
}

func (this *IntermediateGroup) Copy() Operator {
	return &IntermediateGroup{this.base.copy(), this.plan}
}

func (this *IntermediateGroup) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *IntermediateGroup) beforeItems(context *Context, parent value.Value) bool {
	return true
}

func (this *IntermediateGroup) processItem(item value.AnnotatedValue, context *Context) bool {
	return true
}

func (this *IntermediateGroup) afterItems(context *Context) {
}

func NewFinalGroup(plan *plan.FinalGroup) *FinalGroup {
	rv := &FinalGroup{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *FinalGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFinalGroup(this)
}

func (this *FinalGroup) Copy() Operator {
	return &FinalGroup{this.base.copy(), this.plan}
}

func (this *FinalGroup) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *FinalGroup) beforeItems(context *Context, parent value.Value) bool {
	return true
}

func (this *FinalGroup) processItem(item value.AnnotatedValue, context *Context) bool {
	return true
}

func (this *FinalGroup) afterItems(context *Context) {
}
