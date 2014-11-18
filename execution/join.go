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
	kv, e := this.plan.Term().Keys().Evaluate(item, context)
	if e != nil {
		context.Error(errors.NewError(e, "Error evaluating JOIN keys."))
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
		return !this.plan.Outer() || this.sendItem(item)
	}

	// Build list of keys
	keys := make([]string, 0, len(acts))
	for _, key := range acts {
		k := value.NewValue(key).Actual()
		switch k := k.(type) {
		case string:
			keys = append(keys, k)
		}
	}

	// Fetch
	pairs, err := this.plan.Keyspace().Fetch(keys)
	if err != nil {
		context.Error(err)
		return false
	}

	found := len(pairs) > 0

	// Attach and send
	for i, pair := range pairs {
		joinItem := pair.Value
		var jv value.AnnotatedValue

		// Apply projection, if any
		projection := this.plan.Term().Projection()
		if projection != nil {
			projectedItem, e := projection.Evaluate(joinItem, context)
			if e != nil {
				context.Error(errors.NewError(e,
					"Error evaluating join path."))
				return false
			}

			jv = value.NewAnnotatedValue(projectedItem)
		} else {
			jv = value.NewAnnotatedValue(joinItem)
		}

		jv.SetAttachment("meta", map[string]interface{}{"id": keys[i]})

		var av value.AnnotatedValue
		if i < len(pairs)-1 {
			av = value.NewAnnotatedValue(item.Copy())
		} else {
			av = item
		}

		av.SetField(this.plan.Term().Alias(), jv)

		if !this.sendItem(av) {
			return false
		}
	}

	return found || !this.plan.Outer() || this.sendItem(item)
}
