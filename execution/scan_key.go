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

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

// KeyScan is used for KEYS clauses (except after JOIN / NEST).
type KeyScan struct {
	base
	plan *plan.KeyScan
}

func NewKeyScan(plan *plan.KeyScan) *KeyScan {
	rv := &KeyScan{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *KeyScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitKeyScan(this)
}

func (this *KeyScan) Copy() Operator {
	return &KeyScan{this.base.copy(), this.plan}
}

func (this *KeyScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover()       // Recover from any panic
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		keys, e := this.plan.Keys().Evaluate(parent, context)
		if e != nil {
			context.Error(errors.NewEvaluationError(e, "KEYS"))
			return
		}

		actuals := keys.Actual()
		switch actuals.(type) {
		case []interface{}:
		case nil:
			actuals = []interface{}(nil)
		default:
			actuals = []interface{}{actuals}
		}

		acts := actuals.([]interface{})

		for _, key := range acts {
			cv := value.NewScopeValue(make(map[string]interface{}), parent)
			av := value.NewAnnotatedValue(cv)
			av.SetAttachment("meta", map[string]interface{}{"id": key})
			if !this.sendItem(av) {
				break
			}
		}
	})
}

func (this *KeyScan) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
