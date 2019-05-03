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

type CountScan struct {
	base
	plan *plan.CountScan
}

func NewCountScan(plan *plan.CountScan, context *Context) *CountScan {
	rv := &CountScan{
		plan: plan,
	}

	newRedirectBase(&rv.base)
	rv.output = rv
	return rv
}

func (this *CountScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCountScan(this)
}

func (this *CountScan) Copy() Operator {
	rv := &CountScan{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *CountScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		if !this.active() {
			return
		}
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		this.setExecPhase(COUNT, context)
		defer func() { this.switchPhase(_NOTIME) }() // accrue current phase's time
		defer this.notify()                          // Notify that I have stopped

		this.switchPhase(_SERVTIME)
		count, e := this.plan.Keyspace().Count(context)
		this.switchPhase(_EXECTIME)

		if e != nil {
			context.Error(e)
			return
		}

		cv := value.NewScopeValue(nil, parent)
		av := value.NewAnnotatedValue(cv)
		av.SetAttachment("count", value.NewValue(count))
		this.sendItem(av)
	})
}

func (this *CountScan) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
