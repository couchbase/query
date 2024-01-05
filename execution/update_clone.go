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

var _CLONE_OP_POOL util.FastPool

func init() {
	util.NewFastPool(&_CLONE_OP_POOL, func() interface{} {
		return &Clone{}
	})
}

// Enable copy-before-write, so that all reads use old values
type Clone struct {
	base
	plan *plan.Clone
}

func NewClone(plan *plan.Clone, context *Context) *Clone {
	rv := _CLONE_OP_POOL.Get().(*Clone)
	rv.plan = plan

	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *Clone) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitClone(this)
}

func (this *Clone) Copy() Operator {
	rv := _CLONE_OP_POOL.Get().(*Clone)
	rv.plan = this.plan
	this.base.copy(&rv.base)
	return rv
}

func (this *Clone) PlanOp() plan.Operator {
	return this.plan
}

func (this *Clone) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent, nil)
}

func (this *Clone) processItem(item value.AnnotatedValue, context *Context) bool {
	clone := item.CopyForUpdate()
	if av, ok := clone.(value.AnnotatedValue); ok && av != nil {
		options := make(map[string]interface{})
		av.SetMetaField(value.META_EXPIRATION, uint32(0))
		options["xattrs"] = av.GetMetaField(value.META_XATTRS)
		av.SetAttachment(value.ATT_OPTIONS, value.NewValue(options))
	}

	item.SetAttachment(value.ATT_CLONE, clone)
	return this.sendItem(item)
}

func (this *Clone) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *Clone) Done() {
	this.baseDone()
	if this.isComplete() {
		_CLONE_OP_POOL.Put(this)
	}
}
