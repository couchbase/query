//  Copyright 2021-Present Couchbase, Inc.
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

type InferExpression struct {
	base
	plan *plan.InferExpression
}

func NewInferExpression(plan *plan.InferExpression, context *Context) *InferExpression {
	rv := &InferExpression{
		plan: plan,
	}

	newBase(&rv.base, context)
	rv.newStopChannel()
	rv.execPhase = INFER
	rv.output = rv
	return rv
}

func (this *InferExpression) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitInferExpression(this)
}

func (this *InferExpression) Copy() Operator {
	rv := &InferExpression{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *InferExpression) PlanOp() plan.Operator {
	return this.plan
}

func (this *InferExpression) RunOnce(context *Context, parent value.Value) {
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
		go infer.InferExpression(context, this.plan.Node().Expression(), this.plan.Node().With(), conn)

		var val value.Value

		ok := true
		for ok {
			item, cont := this.getItemValue(conn.ValueChannel())
			if item != nil && cont {
				val = item.(value.Value)
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

func (this *InferExpression) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

// send a stop/pause
func (this *InferExpression) SendAction(action opAction) {
	this.chanSendAction(action)
}
