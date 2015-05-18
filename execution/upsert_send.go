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
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type SendUpsert struct {
	base
	plan *plan.SendUpsert
}

func NewSendUpsert(plan *plan.SendUpsert) *SendUpsert {
	rv := &SendUpsert{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *SendUpsert) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSendUpsert(this)
}

func (this *SendUpsert) Copy() Operator {
	return &SendUpsert{this.base.copy(), this.plan}
}

func (this *SendUpsert) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent)
}

func (this *SendUpsert) processItem(item value.AnnotatedValue, context *Context) bool {
	return this.enbatch(item, this, context)
}

func (this *SendUpsert) afterItems(context *Context) {
	this.flushBatch(context)
}

func (this *SendUpsert) flushBatch(context *Context) bool {
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
			// UPSERT ... SELECT
			key, err = keyExpr.Evaluate(av, context)
			if err != nil {
				context.Error(errors.NewEvaluationError(err,
					fmt.Sprintf("UPSERT key for %v", av.GetValue())))
				continue
			}

			if valExpr != nil {
				val, err = valExpr.Evaluate(av, context)
				if err != nil {
					context.Error(errors.NewEvaluationError(err,
						fmt.Sprintf("UPSERT value for %v", av.GetValue())))
					continue
				}
			} else {
				val = av
			}
		} else {
			// UPSERT ... VALUES
			key, ok = av.GetAttachment("key").(value.Value)
			if !ok {
				context.Error(errors.NewUpsertKeyError(av.GetValue()))
				continue
			}

			val, ok = av.GetAttachment("value").(value.Value)
			if !ok {
				context.Error(errors.NewUpsertValueError(av.GetValue()))
				continue
			}
		}

		dpair.Key, ok = key.Actual().(string)
		if !ok {
			context.Error(errors.NewUpsertKeyTypeError(key))
			continue
		}

		dpair.Value = val
		i++
	}

	dpairs = dpairs[0:i]
	this.batch = nil

	timer := time.Now()

	// Perform the actual UPSERT
	keys, e := this.plan.Keyspace().Upsert(dpairs)

	context.AddPhaseTime("upsert", time.Since(timer))

	// Update mutation count with number of upserted docs
	context.AddMutationCount(uint64(len(keys)))

	if e != nil {
		context.Error(e)
	}

	// Capture the upserted keys in case there is a RETURNING clause
	for i, k := range keys {
		av := value.NewAnnotatedValue(make(map[string]interface{}))
		av.SetAttachment("meta", map[string]interface{}{"id": k})
		av.SetField(this.plan.Alias(), dpairs[i].Value)
		if !this.sendItem(av) {
			return false
		}
	}

	return true
}

func (this *SendUpsert) readonly() bool {
	return false
}
