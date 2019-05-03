//  Copyright (c) 2019 Couchbase, Inc.
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
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type ExecuteFunction struct {
	base
	plan *plan.ExecuteFunction
}

func NewExecuteFunction(plan *plan.ExecuteFunction, context *Context) *ExecuteFunction {
	rv := &ExecuteFunction{
		plan: plan,
	}

	newRedirectBase(&rv.base)
	rv.output = rv
	return rv
}

func (this *ExecuteFunction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExecuteFunction(this)
}

func (this *ExecuteFunction) Copy() Operator {
	rv := &ExecuteFunction{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *ExecuteFunction) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		if !this.active() {
			return
		}
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		defer this.notify() // Notify that I have stopped

		if context.Readonly() {
			return
		}

		// evaluate the parameter list
		var args []value.Value

		exprs := this.plan.Expressions()
		l := len(exprs)
		if l > 0 {
			args = make([]value.Value, l)
			for e, _ := range exprs {
				ev, err := exprs[e].Evaluate(parent, context)
				if err != nil {
					context.Error(errors.NewEvaluationError(err, "ExecuteFunction"))
					return
				}
				args[e] = ev
			}
		}

		val, err := functions.ExecuteFunction(this.plan.Name(), functions.NONE, args, context)
		if err != nil {
			context.Error(err)
		} else {
			av := value.NewAnnotatedValue(val)
			this.sendItem(av)
		}
	})
}

func (this *ExecuteFunction) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
