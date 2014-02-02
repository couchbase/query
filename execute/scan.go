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

type EqualScan struct {
	operatorBase
	plan *plan.EqualScan
}

type RangeScan struct {
	operatorBase
	plan *plan.RangeScan
}

type DualScan struct {
	operatorBase
	plan *plan.DualScan
}

type KeyScan struct {
	operatorBase
	plan *plan.KeyScan
}

type ValueScan struct {
	operatorBase
	plan *plan.ValueScan
}

type DummyScan struct {
	operatorBase
}

func NewEqualScan(plan *plan.EqualScan) *EqualScan {
	return &EqualScan{plan: plan}
}

func (this *EqualScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitEqualScan(this)
}

func (this *EqualScan) Copy() Operator {
	return &EqualScan{this.operatorBase.copy(), this.plan}
}

func (this *EqualScan) Run(context *Context, parent value.Value) {
}

func NewRangeScan(plan *plan.RangeScan) *RangeScan {
	return &RangeScan{plan: plan}
}

func (this *RangeScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitRangeScan(this)
}

func (this *RangeScan) Copy() Operator {
	return &RangeScan{this.operatorBase.copy(), this.plan}
}

func (this *RangeScan) Run(context *Context, parent value.Value) {
}

func NewDualScan(plan *plan.DualScan) *DualScan {
	return &DualScan{plan: plan}
}

func (this *DualScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDualScan(this)
}

func (this *DualScan) Copy() Operator {
	return &DualScan{this.operatorBase.copy(), this.plan}
}

func (this *DualScan) Run(context *Context, parent value.Value) {
}

func NewKeyScan(plan *plan.KeyScan) *KeyScan {
	return &KeyScan{plan: plan}
}

func (this *KeyScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitKeyScan(this)
}

func (this *KeyScan) Copy() Operator {
	return &KeyScan{this.operatorBase.copy(), this.plan}
}

func (this *KeyScan) Run(context *Context, parent value.Value) {
}

func NewValueScan(plan *plan.ValueScan) *ValueScan {
	return &ValueScan{plan: plan}
}

func (this *ValueScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitValueScan(this)
}

func (this *ValueScan) Copy() Operator {
	return &ValueScan{this.operatorBase.copy(), this.plan}
}

func (this *ValueScan) Run(context *Context, parent value.Value) {
}

func NewDummyScan() *DummyScan {
	return &DummyScan{}
}

func (this *DummyScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDummyScan(this)
}

func (this *DummyScan) Copy() Operator {
	return &DummyScan{this.operatorBase.copy()}
}

func (this *DummyScan) Run(context *Context, parent value.Value) {
}
