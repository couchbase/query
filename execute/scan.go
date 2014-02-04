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

type FullScan struct {
	base
	plan *plan.FullScan
}

type ParentScan struct {
	base
	plan *plan.ParentScan
}

type EqualScan struct {
	base
	plan *plan.EqualScan
}

type RangeScan struct {
	base
	plan *plan.RangeScan
}

type DualScan struct {
	base
	plan *plan.DualScan
}

type KeyScan struct {
	base
	plan *plan.KeyScan
}

type ValueScan struct {
	base
	plan *plan.ValueScan
}

type DummyScan struct {
	base
}

func NewFullScan(plan *plan.FullScan) *FullScan {
	return &FullScan{
		base: newBase(),
		plan: plan,
	}
}

func (this *FullScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFullScan(this)
}

func (this *FullScan) Copy() Operator {
	return &FullScan{this.base.copy(), this.plan}
}

func (this *FullScan) RunOnce(context *Context, parent value.Value) {
}

func NewParentScan(plan *plan.ParentScan) *ParentScan {
	return &ParentScan{
		base: newBase(),
		plan: plan,
	}
}

func (this *ParentScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitParentScan(this)
}

func (this *ParentScan) Copy() Operator {
	return &ParentScan{this.base.copy(), this.plan}
}

func (this *ParentScan) RunOnce(context *Context, parent value.Value) {
}

func NewEqualScan(plan *plan.EqualScan) *EqualScan {
	return &EqualScan{
		base: newBase(),
		plan: plan,
	}
}

func (this *EqualScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitEqualScan(this)
}

func (this *EqualScan) Copy() Operator {
	return &EqualScan{this.base.copy(), this.plan}
}

func (this *EqualScan) RunOnce(context *Context, parent value.Value) {
}

func NewRangeScan(plan *plan.RangeScan) *RangeScan {
	return &RangeScan{
		base: newBase(),
		plan: plan,
	}
}

func (this *RangeScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitRangeScan(this)
}

func (this *RangeScan) Copy() Operator {
	return &RangeScan{this.base.copy(), this.plan}
}

func (this *RangeScan) RunOnce(context *Context, parent value.Value) {
}

func NewDualScan(plan *plan.DualScan) *DualScan {
	return &DualScan{
		base: newBase(),
		plan: plan,
	}
}

func (this *DualScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDualScan(this)
}

func (this *DualScan) Copy() Operator {
	return &DualScan{this.base.copy(), this.plan}
}

func (this *DualScan) RunOnce(context *Context, parent value.Value) {
}

func NewKeyScan(plan *plan.KeyScan) *KeyScan {
	return &KeyScan{
		base: newBase(),
		plan: plan,
	}
}

func (this *KeyScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitKeyScan(this)
}

func (this *KeyScan) Copy() Operator {
	return &KeyScan{this.base.copy(), this.plan}
}

func (this *KeyScan) RunOnce(context *Context, parent value.Value) {
}

func NewValueScan(plan *plan.ValueScan) *ValueScan {
	return &ValueScan{
		base: newBase(),
		plan: plan,
	}
}

func (this *ValueScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitValueScan(this)
}

func (this *ValueScan) Copy() Operator {
	return &ValueScan{this.base.copy(), this.plan}
}

func (this *ValueScan) RunOnce(context *Context, parent value.Value) {
}

func NewDummyScan() *DummyScan {
	return &DummyScan{
		base: newBase(),
	}
}

func (this *DummyScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDummyScan(this)
}

func (this *DummyScan) Copy() Operator {
	return &DummyScan{this.base.copy()}
}

func (this *DummyScan) RunOnce(context *Context, parent value.Value) {
}
