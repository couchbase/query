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
)

type Join struct {
	operatorBase
	plan *plan.Join
}

type Nest struct {
	operatorBase
	plan *plan.Nest
}

type Unnest struct {
	operatorBase
	plan *plan.Unnest
}

func NewJoin(plan *plan.Join) *Join {
	return &Join{plan: plan}
}

func (this *Join) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitJoin(this)
}

func (this *Join) Copy() Operator {
	return &Join{this.operatorBase.copy(), this.plan}
}

func (this *Join) Run(context *Context) {
}

func NewNest(plan *plan.Nest) *Nest {
	return &Nest{plan: plan}
}

func (this *Nest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitNest(this)
}

func (this *Nest) Copy() Operator {
	return &Nest{this.operatorBase.copy(), this.plan}
}

func (this *Nest) Run(context *Context) {
}

func NewUnnest(plan *plan.Unnest) *Unnest {
	return &Unnest{plan: plan}
}

func (this *Unnest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnnest(this)
}

func (this *Unnest) Copy() Operator {
	return &Unnest{this.operatorBase.copy(), this.plan}
}

func (this *Unnest) Run(context *Context) {
}
