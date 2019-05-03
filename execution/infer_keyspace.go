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

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type InferKeyspace struct {
	base
	plan *plan.InferKeyspace
}

func NewInferKeyspace(plan *plan.InferKeyspace, context *Context) *InferKeyspace {
	rv := &InferKeyspace{
		plan: plan,
	}

	newBase(&rv.base, context)
	rv.newStopChannel()
	rv.execPhase = INFER
	rv.output = rv
	return rv
}

func (this *InferKeyspace) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitInferKeyspace(this)
}

func (this *InferKeyspace) Copy() Operator {
	rv := &InferKeyspace{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *InferKeyspace) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		if !this.active() {
			return
		}
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		defer func() { this.switchPhase(_NOTIME) }()
		defer this.notify() // Notify that I have stopped

		conn := datastore.NewValueConnection(context)
		defer notifyConn(conn.StopChannel())

		using := this.plan.Node().Using()
		infer, err := context.Datastore().Inferencer(using)
		if err != nil {
			context.Error(errors.NewInferencerNotFoundError(err, string(using)))
			return
		}
		go infer.InferKeyspace(this.plan.Keyspace(), this.plan.Node().With(), conn)

		var val value.Value

		ok := true
		for ok {
			item, cont := this.getItemValue(conn.ValueChannel())
			if item != nil && cont {
				val = item.(value.Value)

				// current policy is to only count 'in' documents
				// from operators, not kv
				// add this.addInDocs(1) if this changes
				ok = this.sendItem(value.NewAnnotatedValue(val))
			} else {
				break
			}
		}
	})
}

func (this *InferKeyspace) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

// send a stop
func (this *InferKeyspace) SendStop() {
	this.chanSendStop()
}
