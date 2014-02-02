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

type Offset struct {
	operatorBase
	plan *plan.Offset
}

type Limit struct {
	operatorBase
	plan *plan.Limit
}

func NewOffset(plan *plan.Offset) *Offset {
	return &Offset{plan: plan}
}

func (this *Offset) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitOffset(this)
}

func (this *Offset) Copy() Operator {
	return &Offset{this.operatorBase.copy(), this.plan}
}

func (this *Offset) Run(context *Context, parent value.Value) {
}

func NewLimit(plan *plan.Limit) *Limit {
	return &Limit{plan: plan}
}

func (this *Limit) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitLimit(this)
}

func (this *Limit) Copy() Operator {
	return &Limit{this.operatorBase.copy(), this.plan}
}

func (this *Limit) Run(context *Context, parent value.Value) {
}
