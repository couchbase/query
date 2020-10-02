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
	this.runConsumer(this, context, parent)
}

func (this *Clone) processItem(item value.AnnotatedValue, context *Context) bool {
	clone := item.CopyForUpdate()
	if av, ok := clone.(value.AnnotatedValue); ok && av != nil {
		options := map[string]interface{}{"expiration": uint32(0)}
		mv := av.NewMeta()
		mv["expiration"] = uint32(0)
		options["xattrs"] = mv["xattrs"]
		av.SetAttachment("options", value.NewValue(options))
	}

	item.SetAttachment("clone", clone)
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
