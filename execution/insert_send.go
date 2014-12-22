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

type SendInsert struct {
	base
	plan  *plan.SendInsert
	limit int64
}

func NewSendInsert(plan *plan.SendInsert) *SendInsert {
	rv := &SendInsert{
		base:  newBase(),
		plan:  plan,
		limit: -1,
	}

	rv.output = rv
	return rv
}

func (this *SendInsert) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSendInsert(this)
}

func (this *SendInsert) Copy() Operator {
	return &SendInsert{this.base.copy(), this.plan, this.limit}
}

func (this *SendInsert) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *SendInsert) processItem(item value.AnnotatedValue, context *Context) bool {
	rv := this.limit != 0 && this.enbatch(item, this, context)

	if this.limit > 0 {
		this.limit--
	}

	return rv
}

func (this *SendInsert) beforeItems(context *Context, parent value.Value) bool {
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

func (this *SendInsert) afterItems(context *Context) {
	this.flushBatch(context)
}

func (this *SendInsert) flushBatch(context *Context) bool {
	if len(this.batch) == 0 {
		return true
	}

	keyExpr := this.plan.Key()
	valExpr := this.plan.Value()
	dpairs := make([]datastore.Pair, len(this.batch))
	var key, val value.Value
	var err error
	var ok bool
	i := 0

	for _, av := range this.batch {
		dpair := &dpairs[i]

		if keyExpr != nil {
			// INSERT ... SELECT
			key, err = keyExpr.Evaluate(av, context)
			if err != nil {
				context.Error(errors.NewError(err,
					fmt.Sprintf("Error evaluating INSERT key for %v", av.GetValue())))
				continue
			}

			if valExpr != nil {
				val, err = valExpr.Evaluate(av, context)
				if err != nil {
					context.Error(errors.NewError(err,
						fmt.Sprintf("Error evaluating INSERT value for %v", av.GetValue())))
					continue
				}
			} else {
				val = av
			}
		} else {
			// INSERT ... VALUES
			key, ok = av.GetAttachment("key").(value.Value)
			if !ok {
				context.Error(errors.NewError(nil,
					fmt.Sprintf("No INSERT key for %v", av.GetValue())))
				continue
			}

			val, ok = av.GetAttachment("value").(value.Value)
			if !ok {
				context.Error(errors.NewError(nil,
					fmt.Sprintf("No INSERT value for %v", av.GetValue())))
				continue
			}
		}

		dpair.Key, ok = key.Actual().(string)
		if !ok {
			context.Error(errors.NewError(nil,
				fmt.Sprintf("Cannot INSERT non-string key %v of type %T.", key, key)))
			continue
		}

		dpair.Value = val
		i++
	}

	dpairs = dpairs[0:i]
	this.batch = nil

	// Perform the actual INSERT
	keys, e := this.plan.Keyspace().Insert(dpairs)

	// Update mutation count with number of inserted docs
	context.AddMutationCount(uint64(len(keys)))

	if e != nil {
		context.Error(e)
	}

	// Capture the inserted keys in case there is a RETURNING clause
	for i, k := range keys {
		av := value.NewAnnotatedValue(dpairs[i].Value)
		av.SetAttachment("meta", map[string]interface{}{"id": k})
		if !this.sendItem(av) {
			return false
		}
	}

	return true
}

func (this *SendInsert) readonly() bool {
	return false
}
