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
	_ "fmt"

	"github.com/couchbaselabs/query/err"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

// ValueScan is used when there is a VALUES clause, e.g. in INSERTs.
type ValueScan struct {
	base
	plan *plan.ValueScan
}

func NewValueScan(plan *plan.ValueScan) *ValueScan {
	rv := &ValueScan{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *ValueScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitValueScan(this)
}

func (this *ValueScan) Copy() Operator {
	return &ValueScan{this.base.copy(), this.plan}
}

func (this *ValueScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer close(this.itemChannel) // Broadcast that I have stopped

		vals, e := this.plan.Values().Evaluate(parent, context)
		if e != nil {
			context.ErrorChannel() <- err.NewError(e, "Error evaluating VALUES.")
			return
		}

		actuals := vals.Actual()
		switch actuals.(type) {
		case []interface{}:
		case nil:
			actuals = []interface{}(nil)
		default:
			actuals = []interface{}{actuals}
		}

		acts := actuals.([]interface{})

		for _, act := range acts {
			av := value.NewAnnotatedValue(value.NewValue(act))
			this.output.ItemChannel() <- av
		}
	})
}
