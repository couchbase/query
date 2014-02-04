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

type Filter struct {
	base
	plan *plan.Filter
}

func NewFilter(plan *plan.Filter) *Filter {
	rv := &Filter{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *Filter) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFilter(this)
}

func (this *Filter) Copy() Operator {
	return &Filter{this.base.copy(), this.plan}
}

func (this *Filter) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Filter) processItem(item value.Value, context *Context, parent value.Value) bool {
	return true
}
