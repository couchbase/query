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

// Compute DistinctCount() and Avg().
type FinalGroup struct {
	base
	plan *plan.FinalGroup
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
