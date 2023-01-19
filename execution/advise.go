//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"encoding/json"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type Advise struct {
	base
	plan plan.Operator
}

func NewAdviseIndex(plan plan.Operator, context *Context) *Advise {
	rv := &Advise{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *Advise) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAdvise(this)
}

func (this *Advise) Copy() Operator {
	rv := &Advise{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *Advise) PlanOp() plan.Operator {
	return this.plan
}

func (this *Advise) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic

		active := this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		defer this.notify() // Notify that I have stopped
		if !active {
			return
		}

		bytes, err := this.plan.MarshalJSON()
		if err != nil {
			context.Fatal(errors.NewAdviseIndexError(err, "AdviseIndex: Error marshaling JSON."))
			return
		}

		av := value.NewAnnotatedValue(bytes)
		if context.UseRequestQuota() {
			err := context.TrackValueSize(av.Size())
			if err != nil {
				context.Error(err)
				av.Recycle()
				return
			}
		}
		this.sendItem(av)

	})

}

func (this *Advise) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
		r["advice"] = this.plan
	})
	return json.Marshal(r)
}

func (this *Advise) Done() {
	this.baseDone()
	this.plan = nil
}
