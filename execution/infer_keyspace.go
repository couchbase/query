//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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

func (this *InferKeyspace) PlanOp() plan.Operator {
	return this.plan
}

func (this *InferKeyspace) RunOnce(context *Context, parent value.Value) {
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

		using := this.plan.Node().Using()
		infer, err := context.Datastore().Inferencer(using)
		if err != nil {
			context.Error(errors.NewInferencerNotFoundError(err, string(using)))
			return
		}
		go infer.InferKeyspace(context, this.plan.Keyspace(), this.plan.Node().With(), conn)

		var val value.Value

		ok := true
		for ok {
			item, cont := this.getItemValue(conn.ValueChannel())
			if item != nil && cont {
				val = item.(value.Value)

				// current policy is to only count 'in' documents
				// from operators, not kv
				// add this.addInDocs(1) if this changes
				av := value.NewAnnotatedValue(val)
				if context.UseRequestQuota() {
					err := context.TrackValueSize(av.Size())
					if err != nil {
						context.Error(err)
						av.Recycle()
						return
					}
				}
				ok = this.sendItem(av)
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

// send a stop/pause
func (this *InferKeyspace) SendAction(action opAction) {
	this.chanSendAction(action)
}
