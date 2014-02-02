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

type Precompute struct {
	operatorBase
	plan *plan.Precompute
}

func NewPrecompute(plan *plan.Precompute) *Precompute {
	return &Precompute{plan: plan}
}

func (this *Precompute) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitPrecompute(this)
}

func (this *Precompute) Copy() Operator {
	return &Precompute{this.operatorBase.copy(), this.plan}
}

func (this *Precompute) Run(context *Context) {
}
