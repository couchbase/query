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
	"github.com/couchbaselabs/query/sort"
	"github.com/couchbaselabs/query/value"
)

type Order struct {
	base
	plan    *plan.Order
	values  []value.AnnotatedValue
	length  int
	context *Context
}

const _ORDER_CAP = 1024

func NewOrder(plan *plan.Order) *Order {
	rv := &Order{
		base:   newBase(),
		plan:   plan,
		values: make([]value.AnnotatedValue, _ORDER_CAP),
	}

	rv.output = rv
	return rv
}

func (this *Order) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitOrder(this)
}

func (this *Order) Copy() Operator {
	return &Order{
		base:   this.base.copy(),
		plan:   this.plan,
		values: make([]value.AnnotatedValue, _ORDER_CAP),
	}
}

func (this *Order) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Order) processItem(item value.AnnotatedValue, context *Context) bool {
	if len(this.values) >= this.length {
		values := make([]value.AnnotatedValue, this.length<<1)
		copy(values, this.values)
		this.values = values
	}

	this.values[this.length] = item
	this.length++
	return true
}

func (this *Order) afterItems(context *Context) {
	defer func() { this.values = nil }()

	this.values = this.values[0:this.length]
	this.context = context
	sort.Sort(this)
	this.context = nil

	for _, av := range this.values {
		if !this.sendItem(av) {
			return
		}
	}
}

func (this *Order) Len() int {
	return len(this.values)
}

func (this *Order) Less(i, j int) bool {
	v1 := this.values[i]
	v2 := this.values[j]

	var e1, e2 value.Value
	var c int
	var e error

	for _, term := range this.plan.Terms() {
		e1, e = term.Expression().Evaluate(v1, this.context)
		if e != nil {
			this.context.ErrorChannel() <- errors.NewError(e, "Error evaluating ORDER BY.")
			return false
		}

		e2, e = term.Expression().Evaluate(v2, this.context)
		if e != nil {
			this.context.ErrorChannel() <- errors.NewError(e, "Error evaluating ORDER BY.")
			return false
		}

		c = e1.Collate(e2)

		if c == 0 {
			continue
		} else if term.Descending() {
			return c > 0
		} else {
			return c < 0
		}
	}

	return false
}

func (this *Order) Swap(i, j int) {
	this.values[i], this.values[j] = this.values[j], this.values[i]
}
