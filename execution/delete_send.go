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
	"fmt"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type SendDelete struct {
	base
	plan  *plan.SendDelete
	limit int64
}

func NewSendDelete(plan *plan.SendDelete) *SendDelete {
	rv := &SendDelete{
		base:  newBase(),
		plan:  plan,
		limit: -1,
	}

	rv.output = rv
	return rv
}

func (this *SendDelete) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSendDelete(this)
}

func (this *SendDelete) Copy() Operator {
	return &SendDelete{this.base.copy(), this.plan, this.limit}
}

func (this *SendDelete) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *SendDelete) processItem(item value.AnnotatedValue, context *Context) bool {
	rv := this.limit != 0 && this.enbatch(item, this, context)

	if this.limit > 0 {
		this.limit--
	}

	return rv
}

func (this *SendDelete) beforeItems(context *Context, parent value.Value) bool {
	if this.plan.Limit() == nil {
		return true
	}

	limit, err := this.plan.Limit().Evaluate(parent, context)
	if err != nil {
		context.Error(errors.NewError(err, ""))
		return false
	}

	switch l := limit.Actual().(type) {
	case float64:
		this.limit = int64(l)
	default:
		context.Error(errors.NewError(nil, fmt.Sprintf("Invalid LIMIT %v of type %T.", l, l)))
		return false
	}

	return true
}

func (this *SendDelete) afterItems(context *Context) {
	this.flushBatch(context)
}

func (this *SendDelete) flushBatch(context *Context) bool {
	if len(this.batch) == 0 {
		return true
	}

	keys := make([]string, len(this.batch))

	for i, item := range this.batch {
		dv, ok := item.Field(this.plan.Alias())
		if !ok {
			context.Error(errors.NewError(nil, fmt.Sprintf("DELETE alias %s not found in item.", this.plan.Alias())))
			return false
		}

		av, ok := dv.(value.AnnotatedValue)
		if !ok {
			context.Fatal(errors.NewError(nil, fmt.Sprintf("DELETE alias %s has no metadata in item.", this.plan.Alias())))
			return false
		}

		key, ok := this.requireKey(av, context)
		if !ok {
			return false
		}
		keys[i] = key
	}

	deleted_keys, e := this.plan.Keyspace().Delete(keys)

	// Update mutation count with number of deleted docs:
	context.AddMutationCount(uint64(len(deleted_keys)))

	if e != nil {
		context.Error(e)
	}

	for _, item := range this.batch {
		if !this.sendItem(item) {
			this.batch = nil
			return false
		}
	}

	this.batch = this.batch[:0]
	return true
}

func (this *SendDelete) readonly() bool {
	return false
}
