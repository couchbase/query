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

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

var _SENDINSERT_OP_POOL util.FastPool

func init() {
	util.NewFastPool(&_SENDINSERT_OP_POOL, func() interface{} {
		return &SendInsert{}
	})
}

type SendInsert struct {
	base
	plan     *plan.SendInsert
	keyspace datastore.Keyspace
	limit    int64
}

func NewSendInsert(plan *plan.SendInsert, context *Context) *SendInsert {
	rv := _SENDINSERT_OP_POOL.Get().(*SendInsert)
	rv.plan = plan
	rv.limit = -1
	newBase(&rv.base, context)
	rv.execPhase = INSERT
	rv.output = rv
	return rv
}

func (this *SendInsert) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSendInsert(this)
}

func (this *SendInsert) Copy() Operator {
	rv := _SENDINSERT_OP_POOL.Get().(*SendInsert)
	rv.plan = this.plan
	rv.limit = this.limit
	this.base.copy(&rv.base)
	return rv
}

func (this *SendInsert) PlanOp() plan.Operator {
	return this.plan
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
	this.keyspace = getKeyspace(this.plan.Keyspace(), this.plan.Term().ExpressionTerm(), context)
	if this.keyspace == nil {
		return false
	}

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

func (this *SendInsert) afterItems(context *Context) {
	this.flushBatch(context)
}

func (this *SendInsert) flushBatch(context *Context) bool {
	defer this.releaseBatch(context)

	if len(this.batch) == 0 {
		return true
	}

	var dpairs []value.Pair
	if _INSERT_POOL.Size() >= len(this.batch) {
		dpairs = _INSERT_POOL.Get()
		defer _INSERT_POOL.Put(dpairs)
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
			// INSERT ... SELECT
			key, err = keyExpr.Evaluate(av, context)
			if err != nil {
				context.Error(errors.NewEvaluationError(err,
					fmt.Sprintf("INSERT key for %v", av.GetValue())))
				continue
			}

			if valExpr != nil {
				val, err = valExpr.Evaluate(av, context)
				if err != nil {
					context.Error(errors.NewEvaluationError(err,
						fmt.Sprintf("INSERT value for %v", av.GetValue())))
					continue
				}
			} else {
				val = av
			}

			if optionsExpr != nil {
				options, err = optionsExpr.Evaluate(av, context)
				if err != nil {
					context.Error(errors.NewEvaluationError(err,
						fmt.Sprintf("INSERT value for %v", av.GetValue())))
					continue
				}
			}
		} else {
			// INSERT ... VALUES
			key, ok = av.GetAttachment("key").(value.Value)
			if !ok {
				context.Error(errors.NewInsertKeyError(av.GetValue()))
				continue
			}

			val, ok = av.GetAttachment("value").(value.Value)
			if !ok {
				context.Error(errors.NewInsertValueError(av.GetValue()))
				continue
			}

			options, _ = av.GetAttachment("options").(value.Value)
		}

		dpair.Name, ok = key.Actual().(string)
		if !ok {
			context.Error(errors.NewInsertKeyTypeError(key))
			continue
		}

		if options != nil && options.Type() != value.OBJECT {
			context.Error(errors.NewInsertOptionsTypeError(options))
			continue
		}

		dpair.Options = adjustExpiration(options)
		dpair.Value = this.setDocumentKey(dpair.Name, value.NewAnnotatedValue(val), getExpiration(dpair.Options), context)
		i++
	}

	dpairs = dpairs[0:i]

	this.switchPhase(_SERVTIME)

	// Perform the actual INSERT
	var er errors.Error
	dpairs, er = this.keyspace.Insert(dpairs, context)

	this.switchPhase(_EXECTIME)

	// Update mutation count with number of inserted docs
	context.AddMutationCount(uint64(len(dpairs)))

	if er != nil {
		context.Error(er)
	}

	// Capture the inserted keys in case there is a RETURNING clause
	for _, dp := range dpairs {
		dv := value.NewAnnotatedValue(dp.Value)
		av := value.NewAnnotatedValue(make(map[string]interface{}, 1))
		av.ShareAnnotations(dv)
		av.SetField(this.plan.Alias(), dv)
		if !this.sendItem(av) {
			return false
		}
	}

	return true
}

func (this *SendInsert) readonly() bool {
	return false
}

func (this *SendInsert) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *SendInsert) Done() {
	this.baseDone()
	if this.isComplete() {
		_SENDINSERT_OP_POOL.Put(this)
	}
}

var _INSERT_POOL = value.NewPairPool(_BATCH_SIZE)
