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

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type UpdateStatistics struct {
	base
	plan *plan.UpdateStatistics
}

func NewUpdateStatistics(plan *plan.UpdateStatistics, context *Context) *UpdateStatistics {
	rv := &UpdateStatistics{
		plan: plan,
	}

	newBase(&rv.base, context)
	rv.newStopChannel()
	rv.execPhase = UPDATE_STAT
	rv.output = rv
	return rv
}

func (this *UpdateStatistics) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUpdateStatistics(this)
}

func (this *UpdateStatistics) Copy() Operator {
	rv := &UpdateStatistics{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *UpdateStatistics) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		defer func() { this.switchPhase(_NOTIME) }()
		defer this.notify() // Notify that I have stopped
		if !active {
			return
		}

		conn := datastore.NewValueConnection(context)
		defer notifyConn(conn.StopChannel())

		updstat, err := context.Datastore().StatUpdater()
		if err != nil {
			context.Error(errors.NewStatUpdaterNotFoundError(err))
			return
		}

		go updstat.UpdateStatistics(this.plan.Keyspace(), this.plan.Node().Terms(),
			this.plan.Node().With(), conn, context)

		var val value.Value

		ok := true
		for ok {
			item, cont := this.getItemValue(conn.ValueChannel())
			if item != nil && cont {
				val = item.(value.Value)

				ok = this.sendItem(value.NewAnnotatedValue(val))
			} else {
				break
			}
		}
	})
}

func (this *UpdateStatistics) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

// send a stop/pause
func (this *UpdateStatistics) SendAction(action opAction) {
	this.chanSendAction(action)
}
