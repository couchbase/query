//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execute

import (
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

type SendDelete struct {
	base
	plan *plan.SendDelete
}

func NewSendDelete(plan *plan.SendDelete) *SendDelete {
	rv := &SendDelete{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *SendDelete) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSendDelete(this)
}

func (this *SendDelete) Copy() Operator {
	return &SendDelete{this.base.copy(), this.plan}
}

func (this *SendDelete) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *SendDelete) processItem(item value.AnnotatedValue, context *Context) bool {
	return this.enbatch(item, this, context)
}

func (this *SendDelete) afterItems(context *Context) {
	this.flushBatch(context)
}

func (this *SendDelete) flushBatch(context *Context) bool {
	if len(this.batch) == 0 {
		return true
	}

	keys := make([]string, len(this.batch))

	for i, av := range this.batch {
		key, ok := this.requireKey(av, context)
		if !ok {
			return false
		}
		keys[i] = key
	}

	e := this.plan.Bucket().Delete(keys)
	if e != nil {
		context.ErrorChannel() <- e
		this.batch = nil
		return false
	}

	for _, av := range this.batch {
		if !this.sendItem(av) {
			break
		}
	}

	this.batch = nil
	return true
}
