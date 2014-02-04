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

type ComputeMerge struct {
	base
	plan *plan.ComputeMerge
}

type MergeUpdate struct {
	base
	plan *plan.MergeUpdate
}

type MergeDelete struct {
	base
	plan *plan.MergeDelete
}

type MergeInsert struct {
	base
	plan *plan.MergeInsert
}

type SendMerge struct {
	base
	plan *plan.SendMerge
}

func NewComputeMerge(plan *plan.ComputeMerge) *ComputeMerge {
	return &ComputeMerge{
		base: newBase(),
		plan: plan,
	}
}

func (this *ComputeMerge) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitComputeMerge(this)
}

func (this *ComputeMerge) Copy() Operator {
	return &ComputeMerge{this.base.copy(), this.plan}
}

func (this *ComputeMerge) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *ComputeMerge) processItem(item value.Value, context *Context, parent value.Value) bool {
	return true
}

func (this *ComputeMerge) afterItems(context *Context, parent value.Value) {
}

func NewMergeUpdate(plan *plan.MergeUpdate) *MergeUpdate {
	return &MergeUpdate{
		base: newBase(),
		plan: plan,
	}
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

func (this *MergeUpdate) processItem(item value.Value, context *Context, parent value.Value) bool {
	return true
}

func (this *MergeUpdate) afterItems(context *Context, parent value.Value) {
}

func NewMergeDelete(plan *plan.MergeDelete) *MergeDelete {
	return &MergeDelete{
		base: newBase(),
		plan: plan,
	}
}

func (this *MergeDelete) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitMergeDelete(this)
}

func (this *MergeDelete) Copy() Operator {
	return &MergeDelete{this.base.copy(), this.plan}
}

func (this *MergeDelete) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *MergeDelete) processItem(item value.Value, context *Context, parent value.Value) bool {
	return true
}

func (this *MergeDelete) afterItems(context *Context, parent value.Value) {
}

func NewMergeInsert(plan *plan.MergeInsert) *MergeInsert {
	return &MergeInsert{
		base: newBase(),
		plan: plan,
	}
}

func (this *MergeInsert) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitMergeInsert(this)
}

func (this *MergeInsert) Copy() Operator {
	return &MergeInsert{this.base.copy(), this.plan}
}

func (this *MergeInsert) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *MergeInsert) processItem(item value.Value, context *Context, parent value.Value) bool {
	return true
}

func (this *MergeInsert) afterItems(context *Context, parent value.Value) {
}

func NewSendMerge(plan *plan.SendMerge) *SendMerge {
	return &SendMerge{
		base: newBase(),
		plan: plan,
	}
}

func (this *SendMerge) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSendMerge(this)
}

func (this *SendMerge) Copy() Operator {
	return &SendMerge{this.base.copy(), this.plan}
}

func (this *SendMerge) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *SendMerge) processItem(item value.Value, context *Context, parent value.Value) bool {
	return true
}

func (this *SendMerge) afterItems(context *Context, parent value.Value) {
}
