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

type MergeUpdate struct {
	base
	plan *plan.MergeUpdate
}

func NewMergeUpdate(plan *plan.MergeUpdate) *MergeUpdate {
	rv := &MergeUpdate{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *MergeUpdate) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitMergeUpdate(this)
}

func (this *MergeUpdate) Copy() Operator {
	return &MergeUpdate{this.base.copy(), this.plan}
}

func (this *MergeUpdate) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *MergeUpdate) processItem(item value.AnnotatedValue, context *Context) bool {
	return true
}
