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

// Grouping of input data.
type InitialGroup struct {
	operatorBase
	plan *plan.InitialGroup
}

// Grouping of groups. Recursable.
type IntermediateGroup struct {
	operatorBase
	plan *plan.IntermediateGroup
}

// Compute DistinctCount() and Avg().
type FinalGroup struct {
	operatorBase
	plan *plan.FinalGroup
}

func NewInitialGroup(plan *plan.InitialGroup) *InitialGroup {
	return &InitialGroup{plan: plan}
}

func (this *InitialGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitInitialGroup(this)
}

func (this *InitialGroup) Copy() Operator {
	return &InitialGroup{this.operatorBase.copy(), this.plan}
}

func (this *InitialGroup) Run(context algebra.Context) {
}

func NewIntermediateGroup(plan *plan.IntermediateGroup) *IntermediateGroup {
	return &IntermediateGroup{plan: plan}
}

func (this *IntermediateGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIntermediateGroup(this)
}

func (this *IntermediateGroup) Copy() Operator {
	return &IntermediateGroup{this.operatorBase.copy(), this.plan}
}

func (this *IntermediateGroup) Run(context algebra.Context) {
}

func NewFinalGroup(plan *plan.FinalGroup) *FinalGroup {
	return &FinalGroup{plan: plan}
}

func (this *FinalGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFinalGroup(this)
}

func (this *FinalGroup) Copy() Operator {
	return &FinalGroup{this.operatorBase.copy(), this.plan}
}

func (this *FinalGroup) Run(context algebra.Context) {
}
