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

type Join struct {
	base
	plan *plan.Join
}

type Nest struct {
	base
	plan *plan.Nest
}

type Unnest struct {
	base
	plan *plan.Unnest
}

func NewJoin(plan *plan.Join) *Join {
	rv := &Join{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *Join) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitJoin(this)
}

func (this *Join) Copy() Operator {
	return &Join{this.base.copy(), this.plan}
}

func (this *Join) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Join) processItem(item value.Value, context *Context, parent value.Value) bool {
	return true
}

func NewNest(plan *plan.Nest) *Nest {
	rv := &Nest{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *Nest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitNest(this)
}

func (this *Nest) Copy() Operator {
	return &Nest{this.base.copy(), this.plan}
}

func (this *Nest) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Nest) processItem(item value.Value, context *Context, parent value.Value) bool {
	return true
}

func NewUnnest(plan *plan.Unnest) *Unnest {
	rv := &Unnest{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *Unnest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnnest(this)
}

func (this *Unnest) Copy() Operator {
	return &Unnest{this.base.copy(), this.plan}
}

func (this *Unnest) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Unnest) processItem(item value.Value, context *Context, parent value.Value) bool {
	return true
}
