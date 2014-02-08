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

	"github.com/couchbaselabs/query/err"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

type Join struct {
	base
	plan *plan.Join
}

func NewJoin(plan *plan.Join) *Join {
	rv := &Join{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *Join) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitJoin(this)
}

func (this *Join) Copy() Operator {
	return &Join{this.base.copy(), this.plan}
}

func (this *Join) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Join) processItem(item value.AnnotatedValue, context *Context) bool {
	kv, e := this.plan.Term().Right().Keys().Evaluate(item, context)
	if e != nil {
		context.ErrorChannel() <- err.NewError(e, "Error evaluating JOIN keys.")
		return false
	}

	actuals := kv.Actual()
	switch actuals.(type) {
	case []interface{}:
	case nil:
		actuals = []interface{}(nil)
	default:
		actuals = []interface{}{actuals}
	}

	acts := actuals.([]interface{})
	if len(acts) == 0 {
		// Outer join
		return !this.plan.Term().Outer() || this.sendItem(item)
	}

	// Build list of keys
	keys := make([]string, len(acts))
	for i, key := range acts {
		switch key := key.(type) {
		case string:
			keys[i] = key
		default:
			context.ErrorChannel() <- err.NewError(nil, fmt.Sprintf(
				"Missing or invalid join key %v of type %T.",
				key, key))
			return false
		}
	}

	// Fetch
	joinItems, er := this.plan.Bucket().Fetch(keys)
	if er != nil {
		context.ErrorChannel() <- er
		return false
	}

	// Attach and send
	for i, joinItem := range joinItems {
		// Apply projection, if any
		project := this.plan.Term().Right().Project()
		if project != nil {
			var e error
			joinItem, e = project.Evaluate(joinItem, context)
			if e != nil {
				context.ErrorChannel() <- err.NewError(e,
					"Error evaluating join path.")
				return false
			}
		}

		jv := value.NewAnnotatedValue(joinItem)
		jv.SetAttachment("meta", map[string]interface{}{"id": keys[i]})

		var av value.AnnotatedValue
		if i < len(joinItems)-1 {
			av = item.Copy().(value.AnnotatedValue)
		} else {
			av = item
		}

		av.SetField(this.plan.Alias(), jv)

		if !this.sendItem(av) {
			return false
		}
	}

	return true
}
