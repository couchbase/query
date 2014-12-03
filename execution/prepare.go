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

	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

type Prepare struct {
	base
	plan plan.Operator
}

func NewPrepare(plan plan.Operator) *Prepare {
	rv := &Prepare{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *Prepare) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitPrepare(this)
}

func (this *Prepare) Copy() Operator {
	return &Prepare{this.base.copy(), this.plan}
}

func (this *Prepare) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover()       // Recover from any panic
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		bytes, err := json.Marshal(this.plan)
		if err != nil {
			context.Fatal(errors.NewError(err, "Failed to marshal JSON."))
			return
		}

		value := value.NewAnnotatedValue(bytes)
		this.sendItem(value)

	})
}
