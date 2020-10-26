//  Copyright (c) 2018 Couchbase, Inc.
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

type With struct {
	base
	plan  *plan.With
	child Operator
}

func NewWith(plan *plan.With, context *Context, child Operator) *With {
	rv := &With{
		plan:  plan,
		child: child,
	}

	newBase(&rv.base, context)
	rv.base.setInline()
	rv.output = rv
	return rv
}

func (this *With) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitWith(this)
}

func (this *With) Copy() Operator {
	rv := &With{plan: this.plan, child: this.child.Copy()}
	this.base.copy(&rv.base)
	return rv
}

func (this *With) PlanOp() plan.Operator {
	return this.plan
}

func (this *With) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		this.SetKeepAlive(1, context) // terminate early
		this.switchPhase(_EXECTIME)
		this.setExecPhase(RUN, context)
		defer func() { this.switchPhase(_NOTIME) }() // accrue current phase's time

		if !active || !context.assert(this.child != nil, "With has no child") {
			this.notify()
			this.fail(context)
			return
		}

		this.child.SetInput(this.input)
		this.child.SetOutput(this.output)
		this.child.SetStop(nil)
		this.child.SetParent(this)

		var wv value.AnnotatedValue

		if parent != nil {
			wv = value.NewAnnotatedValue(parent.Copy())
		} else {
			wv = value.NewAnnotatedValue(make(map[string]interface{}, 1))
		}

		for _, b := range this.plan.Bindings() {
			v, e := b.Expression().Evaluate(wv, context)
			if e != nil {
				context.Error(errors.NewEvaluationError(e, "WITH"))
				this.notify()

				// MB-31605 have to start the child for the output and stop
				// operators to be set properly by sequences
				break
			}

			wv.SetField(b.Variable(), v)
		}

		this.fork(this.child, context, wv)
	})
}

func (this *With) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	r["~child"] = this.child
	return json.Marshal(r)
}

func (this *With) accrueTimes(o Operator) {
	if baseAccrueTimes(this, o) {
		return
	}
	copy, _ := o.(*With)
	this.child.accrueTimes(copy.child)
}

func (this *With) SendAction(action opAction) {
	rv := this.baseSendAction(action)
	child := this.child
	if rv && child != nil {
		child.SendAction(action)
	}
}

func (this *With) reopen(context *Context) bool {
	rv := this.baseReopen(context)
	if rv && this.child != nil {
		rv = this.child.reopen(context)
	}
	return rv
}

func (this *With) Done() {
	this.baseDone()
	if this.child != nil {
		child := this.child
		this.child = nil
		child.Done()
	}
}
