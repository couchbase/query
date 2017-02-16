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

type Discard struct {
	base
	plan *plan.Discard
}

func NewDiscard(plan *plan.Discard, context *Context) *Discard {
	rv := &Discard{
		base: newRedirectBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *Discard) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDiscard(this)
}

func (this *Discard) Copy() Operator {
	return &Discard{
		this.base.copy(),
		this.plan,
	}
}

func (this *Discard) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Discard) processItem(item value.AnnotatedValue, context *Context) bool {
	return true
}

func (this *Discard) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
