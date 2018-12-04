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
	"encoding/json"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type Let struct {
	base
	plan *plan.Let
}

func NewLet(plan *plan.Let, context *Context) *Let {
	rv := &Let{
		plan: plan,
	}

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *Let) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitLet(this)
}

func (this *Let) Copy() Operator {
	rv := &Let{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *Let) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Let) processItem(item value.AnnotatedValue, context *Context) bool {
	lv := item.Copy().(value.AnnotatedValue)
	for _, b := range this.plan.Bindings() {
		v, e := b.Expression().Evaluate(lv, context)
		if e != nil {
			context.Error(errors.NewEvaluationError(e, "LET"))
			return false
		}

		lv.SetField(b.Variable(), v)
	}

	item.Recycle()
	return this.sendItem(lv)
}

func (this *Let) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
