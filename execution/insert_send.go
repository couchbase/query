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

	"github.com/couchbaselabs/query/catalog"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

type SendInsert struct {
	base
	plan *plan.SendInsert
}

func NewSendInsert(plan *plan.SendInsert) *SendInsert {
	rv := &SendInsert{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *SendInsert) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSendInsert(this)
}

func (this *SendInsert) Copy() Operator {
	return &SendInsert{this.base.copy(), this.plan}
}

func (this *SendInsert) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *SendInsert) processItem(item value.AnnotatedValue, context *Context) bool {
	return this.enbatch(item, this, context)
}

func (this *SendInsert) afterItems(context *Context) {
	this.flushBatch(context)
}

func (this *SendInsert) flushBatch(context *Context) bool {
	if len(this.batch) == 0 {
		return true
	}

	key := this.plan.Key()
	pairs := make([]catalog.Pair, len(this.batch))
	i := 0

	for _, av := range this.batch {
		pair := &pairs[i]

		// Evaluate and set the key, if any
		if key != nil {
			k, e := key.Evaluate(av, context)
			if e != nil {
				context.WarningChannel() <- errors.NewError(e,
					fmt.Sprintf("Error evaluating INSERT key for value %v.", av.GetValue()))
				continue
			}

			switch k := k.Actual().(type) {
			case string:
				pair.Key = k
			default:
				context.WarningChannel() <- errors.NewError(nil,
					fmt.Sprintf("Unable to INSERT non-string key %v of type %T.", k, k))
				continue
			}
		}

		pair.Value = av
		i++
	}

	pairs = pairs[0:i]
	this.batch = nil

	// Perform the actual INSERT
	keys, e := this.plan.Keyspace().Insert(pairs)
	if e != nil {
		context.ErrorChannel() <- e
		return false
	}

	// Capture the inserted keys in case there's a RETURNING clause
	for i, k := range keys {
		av := pairs[i].Value.(value.AnnotatedValue)
		av.SetAttachment("meta", map[string]interface{}{"id": k})
		if !this.sendItem(av) {
			return false
		}
	}

	return true
}
