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
	"fmt"
	"math"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type Offset struct {
	base
	plan   *plan.Offset
	offset int64
}

func NewOffset(plan *plan.Offset, context *Context) *Offset {
	rv := &Offset{
		base: newBase(context),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *Offset) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitOffset(this)
}

func (this *Offset) Copy() Operator {
	return &Offset{this.base.copy(), this.plan, 0}
}

func (this *Offset) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Offset) beforeItems(context *Context, parent value.Value) bool {
	val, e := this.plan.Expression().Evaluate(parent, context)
	if e != nil {
		context.Error(errors.NewEvaluationError(e, "OFFSET"))
		return false
	}

	actual := val.Actual()
	switch actual := actual.(type) {
	case float64:
		if math.Trunc(actual) == actual {
			this.offset = int64(actual)
			return true
		}
	}

	context.Error(errors.NewInvalidValueError(
		fmt.Sprintf("Invalid OFFSET value %v.", actual)))
	return false
}

func (this *Offset) processItem(item value.AnnotatedValue, context *Context) bool {
	if this.offset > 0 {
		this.offset--
		return true
	} else {
		return this.sendItem(item)
	}
}

func (this *Offset) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
