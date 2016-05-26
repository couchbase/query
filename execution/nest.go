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
	"time"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type Nest struct {
	base
	plan *plan.Nest
}

func NewNest(plan *plan.Nest) *Nest {
	rv := &Nest{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *Nest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitNest(this)
}

func (this *Nest) Copy() Operator {
	return &Nest{
		base: this.base.copy(),
		plan: this.plan,
	}
}

func (this *Nest) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
	t := this.duration - this.chanTime
	context.AddPhaseTime("nest", t)
	this.plan.AddTime(t)
}

func (this *Nest) processItem(item value.AnnotatedValue, context *Context) bool {
	kv, e := this.plan.Term().Keys().Evaluate(item, context)
	if e != nil {
		context.Error(errors.NewEvaluationError(e, "NEST keys"))
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
		// Outer nest
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

	timer := time.Now()

	// Fetch
	pairs, errs := this.plan.Keyspace().Fetch(keys)

	this.duration += time.Since(timer)

	fetchOk := true
	for _, err := range errs {
		context.Error(err)
		if err.IsFatal() {
			fetchOk = false
		}
	}

	if len(pairs) == 0 {
		if this.plan.Outer() {
			item.SetField(this.plan.Term().Alias(), value.EMPTY_ARRAY_VALUE)
			return this.sendItem(item) && fetchOk
		} else {
			return true
		}
	}

	nvs := make([]interface{}, 0, len(pairs))
	for _, pair := range pairs {
		nvs = append(nvs, pair.Value)
	}

	// Attach and send
	item.SetField(this.plan.Term().Alias(), nvs)
	return this.sendItem(item) && fetchOk
}
