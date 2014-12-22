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

	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

// Send to keyspace
type SendUpdate struct {
	base
	plan  *plan.SendUpdate
	limit int64
}

func NewSendUpdate(plan *plan.SendUpdate) *SendUpdate {
	rv := &SendUpdate{
		base:  newBase(),
		plan:  plan,
		limit: -1,
	}

	rv.output = rv
	return rv
}

func (this *SendUpdate) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSendUpdate(this)
}

func (this *SendUpdate) Copy() Operator {
	return &SendUpdate{this.base.copy(), this.plan, this.limit}
}

func (this *SendUpdate) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *SendUpdate) processItem(item value.AnnotatedValue, context *Context) bool {
	rv := this.limit != 0 && this.enbatch(item, this, context)

	if this.limit > 0 {
		this.limit--
	}

	return rv
}

func (this *SendUpdate) beforeItems(context *Context, parent value.Value) bool {
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

func (this *SendUpdate) afterItems(context *Context) {
	this.flushBatch(context)
}

func (this *SendUpdate) flushBatch(context *Context) bool {
	if len(this.batch) == 0 {
		return true
	}

	pairs := make([]datastore.Pair, len(this.batch))

	for i, av := range this.batch {
		key, ok := this.requireKey(av, context)
		if !ok {
			return false
		}

		pairs[i].Key = key
		clone := av.GetAttachment("clone")
		switch clone := clone.(type) {
		case value.AnnotatedValue:
			pairs[i].Value = this.unwrapTop(clone)
		default:
			context.Error(errors.NewError(nil, fmt.Sprintf(
				"Invalid UPDATE value of type %T.", clone)))
			return false
		}
	}

	pairs, e := this.plan.Keyspace().Update(pairs)

	// Update mutation count with number of updated docs
	context.AddMutationCount(uint64(len(pairs)))

	if e != nil {
		context.Error(e)
	}

	for _, av := range this.batch {
		p := av.GetAttachment("clone")
		if !this.sendItem(p.(value.AnnotatedValue)) {
			break
		}
	}

	this.batch = nil
	return true
}

func (this *SendUpdate) readonly() bool {
	return false
}

// The item may have been wrapped in a top-level object by a previous
// operator (key scan, primary scan) - take the inner value if so, as
// we do not want to write the top-level value to the datastore
func (this *SendUpdate) unwrapTop(item value.AnnotatedValue) value.Value {
	top := this.plan.Alias()
	value := item.GetValue()
	inner_value, ok := value.Field(top)

	if ok {
		return inner_value
	}
	return item
}
