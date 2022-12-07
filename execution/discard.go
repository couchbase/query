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

	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type Discard struct {
	base
	plan *plan.Discard
}

func NewDiscard(plan *plan.Discard, context *Context) *Discard {
	rv := &Discard{
		plan: plan,
	}

	newRedirectBase(&rv.base)
	rv.output = rv
	return rv
}

func (this *Discard) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDiscard(this)
}

func (this *Discard) Copy() Operator {
	rv := &Discard{
		plan: this.plan,
	}
	this.base.copy(&rv.base)
	return rv
}

func (this *Discard) PlanOp() plan.Operator {
	return this.plan
}

func (this *Discard) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Discard) processItem(item value.AnnotatedValue, context *Context) bool {

	// item not used past this point
	if context.UseRequestQuota() {
		context.ReleaseValueSize(item.Size())
	}
	item.Recycle()
	return true
}

func (this *Discard) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
