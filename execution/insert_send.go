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
	batchSize   int
}

func NewSendInsert(plan *plan.SendInsert, context *Context) *SendInsert {
	rv := _SENDINSERT_OP_POOL.Get().(*SendInsert)
	rv.plan = plan
	rv.limit = -1
	newBase(&rv.base, context)
	rv.execPhase = INSERT
	rv.output = rv
	rv.batchSize = context.GetPipelineBatch()
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
	this.runConsumer(this, context, parent, nil)
}

func (this *SendInsert) processItem(item value.AnnotatedValue, context *Context) bool {
	rv := this.limit != 0 && this.enbatchSize(item, this, this.batchSize, context, false)

	if this.limit > 0 {
		this.limit--
	}

	return rv
}

func (this *SendInsert) beforeItems(context *Context, parent value.Value) bool {
	this.keyspace = getKeyspace(this.plan.Keyspace(), this.plan.Term().ExpressionTerm(), &this.operatorCtx)
	if this.keyspace == nil {
		return false
	}

	if this.plan.Limit() == nil {
		return true
	}

	lim, err := getLimit(this.plan.Limit(), parent, &this.operatorCtx)
	if err != nil {
		context.Error(err)
		return false
	}

	this.limit = lim
	return true
}

func (this *SendInsert) afterItems(context *Context) {
	this.flushBatch(context)
}

func (this *SendInsert) flushBatch(context *Context) bool {
	defer this.releaseBatch(context)

	fastDiscard := this.plan.FastDiscard()

	curQueue := this.queuedItems()
	if this.batchSize < curQueue {
		defer func() {
			// If sending items downstream, consider downstream op's ValueExchange capacity
			if !fastDiscard {
				size := int(this.output.ValueExchange().cap())
				if curQueue > size {
					curQueue = size
				}
			}
			this.batchSize = curQueue
		}()
	}

	if len(this.batch) == 0 || !this.isRunning() {
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
	var ok, copyOptions bool
	i := 0

	for _, av := range this.batch {
		copyOptions = true
		dpairs = dpairs[0 : i+1]
		dpair := &dpairs[i]
		var key, val, options value.Value

		if keyExpr != nil {
			// INSERT ... SELECT
			key, err = keyExpr.Evaluate(av, &this.operatorCtx)
			if err != nil {
				context.Error(errors.NewEvaluationError(err,
					fmt.Sprintf("INSERT key for %v", av.GetValue())))
				continue
			}

			if valExpr != nil {
				val, err = valExpr.Evaluate(av, &this.operatorCtx)
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
				options, err = optionsExpr.Evaluate(av, &this.operatorCtx)
				if err != nil {
					context.Error(errors.NewEvaluationError(err,
						fmt.Sprintf("INSERT value for %v", av.GetValue())))
					continue
				}
				if optionsExpr.Value() == nil || options.Equals(optionsExpr.Value()) != value.TRUE_VALUE {
					copyOptions = false
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
				context.ReleaseValueSize(av.Size())
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

		dpair.Options = adjustExpiration(options, copyOptions)
		expiration, _ := getExpiration(dpair.Options)
		dpair.Value = this.setDocumentKey(dpair.Name, value.NewAnnotatedValue(val), expiration, context)
		i++
	}

	dpairs = dpairs[0:i]

	this.switchPhase(_SERVTIME)

	// Perform the actual INSERT
	var errs errors.Errors
	var iCount int

	// If there is a RETURNING clause or the index used was #sequentialScan and we need to make the Halloween Problem checks
	preserveMutations := (!fastDiscard || this.plan.SkipNewKeys())

	if preserveMutations {
		skipNewKeys := this.plan.SkipNewKeys()
		for _, dp := range dpairs {
			if skipNewKeys && !context.AddKeyToSkip(dp.Name) {
				return false
			}
		}
	}

	iCount, dpairs, errs = this.keyspace.Insert(dpairs, &this.operatorCtx, preserveMutations)

	// Update mutation count with number of inserted docs
	context.AddMutationCount(uint64(iCount))

	this.switchPhase(_EXECTIME)

	mutationOk := true
	if len(errs) > 0 {
		context.Errors(errs)
		if context.txContext != nil {
			mutationOk = false
		}
	}

	if !fastDiscard {
		for _, dp := range dpairs {
			// Capture the inserted keys in case there is a RETURNING clause
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
	} else {
		for _, item := range this.batch {
			// item not used past this point
			item.Recycle()
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
