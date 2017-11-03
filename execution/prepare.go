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

type Prepare struct {
	base
	plan     *plan.Prepare
	prepared value.Value
}

func NewPrepare(plan *plan.Prepare, context *Context, prepared value.Value) *Prepare {
	rv := &Prepare{
		plan:     plan,
		prepared: prepared,
	}

	newRedirectBase(&rv.base)
	rv.output = rv
	return rv
}

func (this *Prepare) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitPrepare(this)
}

func (this *Prepare) Copy() Operator {
	rv := &Prepare{
		plan:     this.plan,
		prepared: this.prepared,
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *Prepare) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover() // Recover from any panic
		this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		defer this.notify() // Notify that I have stopped
		err := plan.AddPrepared(this.plan.Plan())
		if err != nil {
			context.Fatal(err)
			return
		}
		value := value.NewAnnotatedValue(this.prepared)
		this.sendItem(value)
	})
}

func (this *Prepare) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
