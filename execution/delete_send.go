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

func NewSendDelete(plan *plan.SendDelete, context *Context) *SendDelete {
	rv := &SendDelete{
		base:  newBase(context),
		plan:  plan,
		limit: -1,
	}

	rv.execPhase = DELETE
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
		context.Error(errors.NewEvaluationError(err, "LIMIT clause"))
		return false
	}

	switch l := limit.Actual().(type) {
	case float64:
		this.limit = int64(l)
	default:
		context.Error(errors.NewInvalidValueError(fmt.Sprintf("Invalid LIMIT %v of type %T.", l, l)))
		return false
	}

	return true
}

func (this *SendDelete) afterItems(context *Context) {
	this.flushBatch(context)
}

func (this *SendDelete) flushBatch(context *Context) bool {
	defer this.releaseBatch(context)

	if len(this.batch) == 0 {
		return true
	}

	keys := _STRING_POOL.Get()
	defer _STRING_POOL.Put(keys)

	for _, item := range this.batch {
		dv, ok := item.Field(this.plan.Alias())
		if !ok {
			context.Error(errors.NewDeleteAliasMissingError(this.plan.Alias()))
			return false
		}

		av, ok := dv.(value.AnnotatedValue)
		if !ok {
			context.Error(errors.NewDeleteAliasMetadataError(this.plan.Alias()))
			return false
		}

		key, ok := this.requireKey(av, context)
		if !ok {
			return false
		}

		keys = append(keys, key)
	}

	this.switchPhase(_SERVTIME)

	deleted_keys, e := this.plan.Keyspace().Delete(keys, context)

	this.switchPhase(_EXECTIME)

	// Update mutation count with number of deleted docs:
	context.AddMutationCount(uint64(len(deleted_keys)))

	if e != nil {
		context.Error(e)
	}

	for _, item := range this.batch {
		if !this.sendItem(item) {
			return false
		}
	}

	return true
}

func (this *SendDelete) readonly() bool {
	return false
}

func (this *SendDelete) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
