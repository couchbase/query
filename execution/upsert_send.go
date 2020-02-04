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

type SendUpsert struct {
	base
	plan *plan.SendUpsert
}

func NewSendUpsert(plan *plan.SendUpsert, context *Context) *SendUpsert {
	rv := &SendUpsert{
		plan: plan,
	}

	newBase(&rv.base, context)
	rv.execPhase = UPSERT
	rv.output = rv
	return rv
}

func (this *SendUpsert) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSendUpsert(this)
}

func (this *SendUpsert) Copy() Operator {
	rv := &SendUpsert{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
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
	defer this.releaseBatch(context)

	if len(this.batch) == 0 {
		return true
	}

	var dpairs []value.Pair
	if _UPSERT_POOL.Size() >= len(this.batch) {
		dpairs = _UPSERT_POOL.Get()
		defer _UPSERT_POOL.Put(dpairs)
	} else {
		dpairs = make([]value.Pair, 0, len(this.batch))
	}

	keyExpr := this.plan.Key()
	valExpr := this.plan.Value()
	optionsExpr := this.plan.Options()
	var err error
	var ok bool
	i := 0

	for _, av := range this.batch {
		dpairs = dpairs[0 : i+1]
		dpair := &dpairs[i]
		var key, val, options value.Value

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

			if optionsExpr != nil {
				options, err = optionsExpr.Evaluate(av, context)
				if err != nil {
					context.Error(errors.NewEvaluationError(err,
						fmt.Sprintf("UPSERT value for %v", av.GetValue())))
					continue
				}
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

			options, _ = av.GetAttachment("options").(value.Value)
		}

		dpair.Name, ok = key.Actual().(string)
		if !ok {
			context.Error(errors.NewUpsertKeyTypeError(key))
			continue
		}

		if options != nil && options.Type() != value.OBJECT {
			context.Error(errors.NewInsertOptionsTypeError(options))
			continue
		}

		dpair.Value = val
		dpair.Options = adjustExpiration(options)
		i++
	}

	dpairs = dpairs[0:i]

	this.switchPhase(_SERVTIME)

	// Perform the actual UPSERT
	var er errors.Error
	dpairs, er = this.plan.Keyspace().Upsert(dpairs)

	this.switchPhase(_EXECTIME)

	// Update mutation count with number of upserted docs
	context.AddMutationCount(uint64(len(dpairs)))

	if er != nil {
		context.Error(er)
	}

	// Capture the upserted keys in case there is a RETURNING clause
	for _, dp := range dpairs {
		dv := this.setDocumentKey(dp.Name, value.NewAnnotatedValue(dp.Value), getExpiration(dp.Options), context)
		av := value.NewAnnotatedValue(make(map[string]interface{}, 1))
		av.SetAnnotations(dv)
		av.SetField(this.plan.Alias(), dv)
		if !this.sendItem(av) {
			return false
		}
	}

	return true
}

func (this *SendUpsert) readonly() bool {
	return false
}

func (this *SendUpsert) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

var _UPSERT_POOL = value.NewPairPool(_BATCH_SIZE)
