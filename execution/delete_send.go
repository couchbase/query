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

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

var _SENDDELETE_OP_POOL util.FastPool

func init() {
	util.NewFastPool(&_SENDDELETE_OP_POOL, func() interface{} {
		return &SendDelete{}
	})
}

type SendDelete struct {
	base
	plan      *plan.SendDelete
	keyspace  datastore.Keyspace
	limit     int64
	mk        missingKeys
	batchSize int
}

func NewSendDelete(plan *plan.SendDelete, context *Context) *SendDelete {
	rv := _SENDDELETE_OP_POOL.Get().(*SendDelete)
	rv.plan = plan
	rv.limit = -1
	rv.mk.validate = plan.ValidateKeys()

	newBase(&rv.base, context)
	rv.execPhase = DELETE
	rv.output = rv
	rv.batchSize = context.GetPipelineBatch()
	return rv
}

func (this *SendDelete) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSendDelete(this)
}

func (this *SendDelete) Copy() Operator {
	rv := _SENDDELETE_OP_POOL.Get().(*SendDelete)
	rv.plan = this.plan
	rv.limit = this.limit
	rv.mk.validate = this.mk.validate
	this.base.copy(&rv.base)
	return rv
}

func (this *SendDelete) PlanOp() plan.Operator {
	return this.plan
}

func (this *SendDelete) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent, nil)
}

func (this *SendDelete) processItem(item value.AnnotatedValue, context *Context) bool {
	rv := this.limit != 0 && this.enbatchSize(item, this, this.batchSize, context, false)

	if this.limit > 0 {
		this.limit--
	}

	return rv
}

func (this *SendDelete) beforeItems(context *Context, parent value.Value) bool {
	this.mk.reset()

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

func (this *SendDelete) afterItems(context *Context) {
	this.flushBatch(context)
	this.mk.report(context, this.plan.Alias)
}

func (this *SendDelete) flushBatch(context *Context) bool {
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

	var pairs []value.Pair
	if _DELETE_POOL.Size() >= len(this.batch) {
		pairs = _DELETE_POOL.Get()
		defer _DELETE_POOL.Put(pairs)
	} else {
		pairs = make([]value.Pair, 0, len(this.batch))
	}

	for i, item := range this.batch {
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

		key, ok := this.getDocumentKey(av, context)
		if !ok {
			return false
		}

		pairs = pairs[0 : i+1]
		pair := &pairs[i]
		pair.Name = key
		pair.Value = av
	}

	// If there is a RETURNING clause or USE KEYS VALIDATE clause
	preserveMutations := (!fastDiscard || this.mk.validate)

	this.switchPhase(_SERVTIME)

	dCount, dpairs, errs := this.keyspace.Delete(pairs, &this.operatorCtx, preserveMutations)

	// Update mutation count with number of deleted docs
	context.AddMutationCount(uint64(dCount))

	this.switchPhase(_EXECTIME)

	if this.mk.validate {
		for _, k := range pairs {
			deleted := false
			for _, k2 := range dpairs {
				if k.Name == k2.Name {
					deleted = true
					break
				}
			}
			if !deleted {
				this.mk.add(k.Name)
			}
		}
	}

	mutationOk := true
	if len(errs) > 0 {
		context.Errors(errs)
		if context.txContext != nil {
			mutationOk = false
		}
	}

	if !fastDiscard {
		for _, item := range this.batch {
			if !this.sendItem(item) {
				return false
			}
		}
	} else {
		for _, item := range this.batch {
			// item not used past this point
			if context.UseRequestQuota() {
				context.ReleaseValueSize(item.Size())
			}
			item.Recycle()
		}
	}

	return mutationOk
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

func (this *SendDelete) Done() {
	this.baseDone()
	if this.isComplete() {
		_SENDDELETE_OP_POOL.Put(this)
	}
}

var _DELETE_POOL = value.NewPairPool(_BATCH_SIZE)
