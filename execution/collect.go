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

	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

// Collect subquery results
type Collect struct {
	base
	plan   *plan.Collect
	values []interface{}
}

const _COLLECT_CAP = 64

func NewCollect(plan *plan.Collect, context *Context) *Collect {
	rv := &Collect{
		base:   newBase(context),
		plan:   plan,
		values: make([]interface{}, 0, _COLLECT_CAP),
	}

	rv.output = rv
	return rv
}

func (this *Collect) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCollect(this)
}

func (this *Collect) Copy() Operator {
	return &Collect{
		base:   this.base.copy(),
		plan:   this.plan,
		values: make([]interface{}, 0, _COLLECT_CAP),
	}
}

func (this *Collect) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Collect) processItem(item value.AnnotatedValue, context *Context) bool {
	if len(this.values) == cap(this.values) {
		values := make([]interface{}, len(this.values), len(this.values)<<1)
		copy(values, this.values)
		this.values = values
	}

	this.values = append(this.values, item.Actual())
	return true
}

func (this *Collect) ValuesOnce() value.Value {
	defer this.releaseValues()
	return value.NewValue(this.values)
}

func (this *Collect) releaseValues() {
	this.values = nil
}

func (this *Collect) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
