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
	"github.com/couchbase/query/expression"
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

func (this *UpdateStatistics) PlanOp() plan.Operator {
	return this.plan
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

		if this.plan.Node().Delete() {
			go updstat.DeleteStatistics(this.plan.Keyspace(), this.plan.Node().Terms(),
				conn, context)
		} else {
			var indexes []datastore.Index
			var err1 errors.Error
			if this.plan.Node().IndexAll() || len(this.plan.Node().Indexes()) > 0 {
				indexes, err1 = getIndexes(context, parent, this.plan.Keyspace(),
					this.plan.Node().Indexes(), this.plan.Node().Using())
				if err1 != nil {
					context.Error(err1)
					return
				}
			}
			go updstat.UpdateStatistics(this.plan.Keyspace(), indexes,
				this.plan.Node().Terms(), this.plan.Node().With(), conn, context, false)
		}

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

func getIndexes(context *Context, parent value.Value, keyspace datastore.Keyspace,
	idxExprs expression.Expressions, using datastore.IndexType) ([]datastore.Index, errors.Error) {

	indexer, err := keyspace.Indexer(using)
	if err != nil {
		return nil, err
	}

	var idxNames []string
	if len(idxExprs) > 0 {
		idxNames, err = getIndexNames(context, parent, idxExprs, "update_statistics")
	} else {
		// all indexes
		idxNames, err = indexer.IndexNames()
	}
	if err != nil {
		return nil, err
	}

	ikey := "execution.update_statistics.index_by_name"
	indexes := make([]datastore.Index, 0, len(idxExprs))
	for _, idxName := range idxNames {
		index, err := indexer.IndexByName(idxName)
		if err != nil {
			return nil, errors.NewIndexNotFoundError(idxName, ikey, err)
		}
		if index.IsPrimary() {
			continue
		}
		indexes = append(indexes, index)
	}

	return indexes, nil
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
