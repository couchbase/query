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
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type Stream struct {
	base
	plan        *plan.Stream
	stopContext *Context
}

var _STREAM_OP_POOL util.FastPool

func init() {
	util.NewFastPool(&_STREAM_OP_POOL, func() interface{} {
		return &Stream{}
	})
}

func NewStream(plan *plan.Stream, context *Context) *Stream {
	rv := _STREAM_OP_POOL.Get().(*Stream)
	rv.plan = plan

	// Stream does not run inside a parallel group and is not
	// guaranteed to have a single producer
	if context.MaxParallelism() == 1 {
		newSerializedBase(&rv.base, context)
	} else {
		newRedirectBase(&rv.base, context)
	}
	rv.stopContext = context
	rv.output = rv
	rv.execPhase = STREAM
	return rv
}

func (this *Stream) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitStream(this)
}

func (this *Stream) Copy() Operator {
	rv := _STREAM_OP_POOL.Get().(*Stream)
	rv.plan = this.plan
	rv.stopContext = this.stopContext
	this.base.copy(&rv.base)
	return rv
}

func (this *Stream) PlanOp() plan.Operator {
	return this.plan
}

func (this *Stream) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *Stream) processItem(item value.AnnotatedValue, context *Context) bool {
	ok := context.Result(item)
	if ok {
		this.addOutDocs(1)
	}

	// MB-53235 for serialized operators item management rests with the producer
	if ok || !this.serialized {

		// item not used past this point
		if context.UseRequestQuota() {
			context.ReleaseValueSize(item.Size())
		}
		item.Recycle()
	}
	return ok
}

func (this *Stream) afterItems(context *Context) {
	context.CloseResults()
}

func (this *Stream) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *Stream) SendAction(action opAction) {
	this.baseSendAction(action)

	// always close results on stop if the stream operator didn't get to start
	if action == _ACTION_STOP && this.getBase().opState == _KILLED {
		this.stopContext.CloseResults()
	}
}

func (this *Stream) Done() {
	this.baseDone()
	if this.isComplete() {
		_STREAM_OP_POOL.Put(this)
	}
}
