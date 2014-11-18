//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execution

import (
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

type Unnest struct {
	base
	plan *plan.Unnest
}

func NewUnnest(plan *plan.Unnest) *Unnest {
	rv := &Unnest{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *Unnest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnnest(this)
}

func (this *Unnest) Copy() Operator {
	return &Unnest{this.base.copy(), this.plan}
}

func (this *Unnest) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Unnest) processItem(item value.AnnotatedValue, context *Context) bool {
	ev, err := this.plan.Term().Expression().Evaluate(item, context)
	if err != nil {
		context.Error(errors.NewError(err, "Error evaluating UNNEST path."))
		return false
	}

	actuals := ev.Actual()
	switch actuals.(type) {
	case []interface{}:
	case nil:
		actuals = []interface{}(nil)
	default:
		actuals = []interface{}{actuals}
	}

	acts := actuals.([]interface{})
	if len(acts) == 0 {
		// Outer unnest
		return !this.plan.Term().Outer() || this.sendItem(item)
	}

	// Attach and send
	for i, act := range acts {
		var av value.AnnotatedValue
		if i < len(acts)-1 {
			av = value.NewAnnotatedValue(item.Copy())
		} else {
			av = item
		}

		av.SetField(this.plan.Alias(), value.NewValue(act))

		if !this.sendItem(av) {
			return false
		}
	}

	return true
}
