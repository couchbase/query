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

// Enable copy-before-write, so that all reads use old values
type Clone struct {
	base
}

// Write to copy
type Set struct {
	base
	plan *plan.Set
}

// Write to copy
type Unset struct {
	base
	plan *plan.Unset
}

// Send to bucket
type SendUpdate struct {
	base
	plan *plan.SendUpdate
}

func NewClone() *Clone {
	return &Clone{
		base: newBase(),
	}
}

func (this *Clone) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitClone(this)
}

func (this *Clone) Copy() Operator {
	return &Clone{this.base.copy()}
}

func (this *Clone) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Clone) processItem(item value.Value, context *Context, parent value.Value) bool {
	return true
}

func (this *Clone) afterItems(context *Context, parent value.Value) {
}

func NewSet(plan *plan.Set) *Set {
	return &Set{
		base: newBase(),
		plan: plan,
	}
}

func (this *Set) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSet(this)
}

func (this *Set) Copy() Operator {
	return &Set{this.base.copy(), this.plan}
}

func (this *Set) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Set) processItem(item value.Value, context *Context, parent value.Value) bool {
	return true
}

func (this *Set) afterItems(context *Context, parent value.Value) {
}

func NewUnset(plan *plan.Unset) *Unset {
	return &Unset{
		base: newBase(),
		plan: plan,
	}
}

func (this *Unset) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnset(this)
}

func (this *Unset) Copy() Operator {
	return &Unset{this.base.copy(), this.plan}
}

func (this *Unset) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Unset) processItem(item value.Value, context *Context, parent value.Value) bool {
	return true
}

func (this *Unset) afterItems(context *Context, parent value.Value) {
}

func NewSendUpdate(plan *plan.SendUpdate) *SendUpdate {
	return &SendUpdate{
		base: newBase(),
		plan: plan,
	}
}

func (this *SendUpdate) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSendUpdate(this)
}

func (this *SendUpdate) Copy() Operator {
	return &SendUpdate{this.base.copy(), this.plan}
}

func (this *SendUpdate) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *SendUpdate) processItem(item value.Value, context *Context, parent value.Value) bool {
	return true
}

func (this *SendUpdate) afterItems(context *Context, parent value.Value) {
}
