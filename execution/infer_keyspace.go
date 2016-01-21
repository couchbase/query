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
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type InferKeyspace struct {
	base
	plan *plan.InferKeyspace
}

func NewInferKeyspace(plan *plan.InferKeyspace) *InferKeyspace {
	rv := &InferKeyspace{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *InferKeyspace) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitInferKeyspace(this)
}

func (this *InferKeyspace) Copy() Operator {
	return &InferKeyspace{this.base.copy(), this.plan}
}

func (this *InferKeyspace) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover()       // Recover from any panic
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		conn := datastore.NewValueConnection(context)
		defer notifyConn(conn.StopChannel())

		var duration time.Duration
		timer := time.Now()
		defer context.AddPhaseTime("InferKeySpace", time.Since(timer)-duration)

		infer, err := context.Datastore().Inferencer(this.plan.Node().Using())
		if err != nil {
			context.Error(errors.NewError(err, "Failed to get Inferencer"))
			return
		}
		go infer.InferKeyspace(this.plan.Keyspace(), this.plan.Node().With(), conn)

		var val value.Value

		ok := true
		for ok {
			select {
			case <-this.stopChannel:
				return
			default:
			}

			select {
			case val, ok = <-conn.ValueChannel():
				if ok {
					t := time.Now()
					ok = this.sendItem(value.NewAnnotatedValue(val))
					duration += time.Since(t)
				}
			case <-this.stopChannel:
				return
			}
		}
	})
}
