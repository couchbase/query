//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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
	plan        *plan.SendInsert
	keyspace    datastore.Keyspace
	limit       int64
	skipNewKeys bool
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
	rv.skipNewKeys = this.skipNewKeys
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

			if context.UseRequestQuota() {
				context.ReleaseValueSize(av.Size())
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
			if context.UseRequestQuota() {
				context.ReleaseValueSize(key.Size() + val.Size())
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
		expiration, _ := getExpiration(dpair.Options)
		dpair.Value = this.setDocumentKey(dpair.Name, value.NewAnnotatedValue(val), expiration, context)
		i++
	}

	dpairs = dpairs[0:i]

	this.switchPhase(_SERVTIME)

	// Perform the actual INSERT
	var errs errors.Errors
	dpairs, errs = this.keyspace.Insert(dpairs, context)

	this.switchPhase(_EXECTIME)

	// Update mutation count with number of inserted docs
	context.AddMutationCount(uint64(len(dpairs)))

	mutationOk := true
	if len(errs) > 0 {
		context.Errors(errs)
		if context.txContext != nil {
			mutationOk = false
		}
	}

	// Capture the inserted keys in case there is a RETURNING clause
	skipNewKeys := this.plan.SkipNewKeys()
	for _, dp := range dpairs {
		if skipNewKeys && !context.AddKeyToSkip(dp.Name) {
			return false
		}
		dv := value.NewAnnotatedValue(dp.Value)
		av := value.NewAnnotatedValue(make(map[string]interface{}, 1))
		av.ShareAnnotations(dv)
		av.SetField(this.plan.Alias(), dv)

		if context.UseRequestQuota() {
			err := context.TrackValueSize(av.Size())
			if err != nil {
				context.Error(err)
				av.Recycle()
				return false
			}
		}
		if !this.sendItem(av) {
			return false
		}
	}

	return mutationOk
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
