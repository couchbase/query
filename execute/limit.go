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
	"fmt"
	"math"

	"github.com/couchbaselabs/query/err"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

type Limit struct {
	base
	plan  *plan.Limit
	limit uint64
}

func NewLimit(plan *plan.Limit) *Limit {
	rv := &Limit{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *Limit) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitLimit(this)
}

func (this *Limit) Copy() Operator {
	return &Limit{this.base.copy(), this.plan, 0}
}

func (this *Limit) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Limit) beforeItems(context *Context, parent value.Value) bool {
	val, e := this.plan.Expression().Evaluate(parent, context)
	if e != nil {
		context.ErrorChannel() <- err.NewError(e, "Error evaluating LIMIT.")
		return false
	}

	actual := val.Actual()
	switch actual := actual.(type) {
	case float64:
		if math.Trunc(actual) == actual {
			this.limit = uint64(actual)
			return true
		}
	}

	context.ErrorChannel() <- err.NewError(nil, fmt.Sprintf("Invalid LIMIT value %v.", actual))
	return false
}

func (this *Limit) processItem(item value.Value, context *Context, parent value.Value) bool {
	if this.limit > 0 {
		this.limit--
		return this.sendItem(item)
	} else {
		return false
	}
}
