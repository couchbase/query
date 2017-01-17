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
	_ "fmt"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
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
		defer context.Recover() // Recover from any panic
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		pairs := this.plan.Values()

		for _, pair := range pairs {
			key, err := pair.Key.Evaluate(parent, context)
			if err != nil {
				context.Error(errors.NewEvaluationError(err, "VALUES"))
				return
			}

			val, err := pair.Value.Evaluate(parent, context)
			if err != nil {
				context.Error(errors.NewEvaluationError(err, "VALUES"))
				return
			}

			av := value.NewAnnotatedValue(nil)
			av.SetAttachment("key", key)
			av.SetAttachment("value", val)

			if !this.sendItem(av) {
				return
			}
		}
	})
}

func (this *ValueScan) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
