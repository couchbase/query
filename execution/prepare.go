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

func NewPrepare(plan *plan.Prepare, prepared value.Value) *Prepare {
	rv := &Prepare{
		base:     newBase(),
		plan:     plan,
		prepared: prepared,
	}

	rv.output = rv
	return rv
}

func (this *Prepare) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitPrepare(this)
}

func (this *Prepare) Copy() Operator {
	return &Prepare{
		this.base.copy(),
		this.plan,
		this.prepared,
	}
}

func (this *Prepare) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover()       // Recover from any panic
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped
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

func (this *Prepare) Done() {
}
