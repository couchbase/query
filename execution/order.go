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
	"github.com/couchbase/query/sort"
	"github.com/couchbase/query/value"
)

type Order struct {
	base
	plan    *plan.Order
	values  value.AnnotatedValues
	context *Context
	terms   []string
}

const _ORDER_CAP = 1024

var _ORDER_POOL = value.NewAnnotatedPool(_ORDER_CAP)

func NewOrder(plan *plan.Order) *Order {
	rv := &Order{
		base:   newBase(),
		plan:   plan,
		values: _ORDER_POOL.Get(),
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
		values: _ORDER_POOL.Get(),
	}
}

func (this *Order) RunOnce(context *Context, parent value.Value) {
	defer this.releaseValues()
	context.AddPhaseOperator(SORT)
	this.runConsumer(this, context, parent)
}

func (this *Order) processItem(item value.AnnotatedValue, context *Context) bool {
	if len(this.values) == cap(this.values) {
		values := make(value.AnnotatedValues, len(this.values), len(this.values)<<1)
		copy(values, this.values)
		this.releaseValues()
		this.values = values
	}

	this.values = append(this.values, item)
	return true
}

func (this *Order) setupTerms(context *Context) {
	this.context = context
	this.terms = make([]string, len(this.plan.Terms()))
	for i, term := range this.plan.Terms() {
		this.terms[i] = term.Expression().String()
	}
}

func (this *Order) afterItems(context *Context) {
	defer this.releaseValues()
	defer func() {
		this.context = nil
		this.terms = nil
	}()

	this.setupTerms(context)
	timer := time.Now()
	sort.Sort(this)
	t := time.Since(timer)
	context.AddPhaseTime(SORT, t)
	this.plan.AddTime(t)

	context.SetSortCount(uint64(this.Len()))
	context.AddPhaseCount(SORT, uint64(this.Len()))

	for _, av := range this.values {
		if !this.sendItem(av) {
			return
		}
	}
}

func (this *Order) releaseValues() {
	_ORDER_POOL.Put(this.values)
	this.values = nil
}

func (this *Order) Len() int {
	return len(this.values)
}

func (this *Order) Less(i, j int) bool {
	return this.lessThan(this.values[i], this.values[j])
}

func (this *Order) lessThan(v1 value.AnnotatedValue, v2 value.AnnotatedValue) bool {
	var ev1, ev2 value.Value
	var c int
	var e error

	for i, term := range this.plan.Terms() {
		s := this.terms[i]

		sv1 := v1.GetAttachment(s)
		switch sv1 := sv1.(type) {
		case value.Value:
			ev1 = sv1
		default:
			ev1, e = term.Expression().Evaluate(v1, this.context)
			if e != nil {
				this.context.Error(errors.NewEvaluationError(e, "ORDER BY"))
				return false
			}

			v1.SetAttachment(s, ev1)
		}

		sv2 := v2.GetAttachment(s)
		switch sv2 := sv2.(type) {
		case value.Value:
			ev2 = sv2
		default:
			ev2, e = term.Expression().Evaluate(v2, this.context)
			if e != nil {
				this.context.Error(errors.NewEvaluationError(e, "ORDER BY"))
				return false
			}

			v2.SetAttachment(s, ev2)
		}

		c = ev1.Collate(ev2)

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
