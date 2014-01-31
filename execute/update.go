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

	"github.com/couchbaselabs/query/algebra"
	"github.com/couchbaselabs/query/plan"
)

// Enable copy-before-write, so that all reads use old values
type Clone struct {
	operatorBase
}

// Write to copy
type Set struct {
	operatorBase
	plan *plan.Set
}

// Write to copy
type Unset struct {
	operatorBase
	plan *plan.Unset
}

// Send to bucket
type SendUpdate struct {
	operatorBase
	plan *plan.SendUpdate
}

func NewClone() *Clone {
	return &Clone{}
}

func (this *Clone) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitClone(this)
}

func (this *Clone) Copy() Operator {
	return &Clone{this.operatorBase.copy()}
}

func (this *Clone) Run(context algebra.Context) {
}

func NewSet(plan *plan.Set) *Set {
	return &Set{plan: plan}
}

func (this *Set) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSet(this)
}

func (this *Set) Copy() Operator {
	return &Set{this.operatorBase.copy(), this.plan}
}

func (this *Set) Run(context algebra.Context) {
}

func NewUnset(plan *plan.Unset) *Unset {
	return &Unset{plan: plan}
}

func (this *Unset) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnset(this)
}

func (this *Unset) Copy() Operator {
	return &Unset{this.operatorBase.copy(), this.plan}
}

func (this *Unset) Run(context algebra.Context) {
}

func NewSendUpdate(plan *plan.SendUpdate) *SendUpdate {
	return &SendUpdate{plan: plan}
}

func (this *SendUpdate) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSendUpdate(this)
}

func (this *SendUpdate) Copy() Operator {
	return &SendUpdate{this.operatorBase.copy(), this.plan}
}

func (this *SendUpdate) Run(context algebra.Context) {
}
