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
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

var _SENDUPDATE_OP_POOL util.FastPool

func init() {
	util.NewFastPool(&_SENDUPDATE_OP_POOL, func() interface{} {
		return &SendUpdate{}
	})
}

// Send to keyspace
type SendUpdate struct {
	base
	plan      *plan.SendUpdate
	keyspace  datastore.Keyspace
	limit     int64
	batchSize int
}

func NewSendUpdate(plan *plan.SendUpdate, context *Context) *SendUpdate {
	rv := _SENDUPDATE_OP_POOL.Get().(*SendUpdate)
	rv.plan = plan
	rv.limit = -1

	newBase(&rv.base, context)
	rv.execPhase = UPDATE
	rv.output = rv
	rv.batchSize = context.GetPipelineBatch()
	return rv
}

func (this *SendUpdate) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSendUpdate(this)
}

func (this *SendUpdate) Copy() Operator {
	rv := _SENDUPDATE_OP_POOL.Get().(*SendUpdate)
	rv.plan = this.plan
	rv.limit = this.limit
	this.base.copy(&rv.base)
	return rv
}

func (this *SendUpdate) PlanOp() plan.Operator {
	return this.plan
}

func (this *SendUpdate) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent, nil)
}

func (this *SendUpdate) processItem(item value.AnnotatedValue, context *Context) bool {
	rv := this.limit != 0 && this.enbatchSize(item, this, this.batchSize, context, false)

	if this.limit > 0 {
		this.limit--
	}

	return rv
}

func (this *SendUpdate) beforeItems(context *Context, parent value.Value) bool {
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

func (this *SendUpdate) afterItems(context *Context) {
	this.flushBatch(context)
}

func (this *SendUpdate) flushBatch(context *Context) bool {
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

	if len(this.batch) == 0 || !this.isRunning() || this.stopped {
		return true
	}

	var pairs []value.Pair
	if _UPDATE_POOL.Size() >= len(this.batch) {
		pairs = _UPDATE_POOL.Get()
		defer _UPDATE_POOL.Put(pairs)
	} else {
		pairs = make([]value.Pair, 0, len(this.batch))
	}

	for i, item := range this.batch {
		if this.stopped {
			return false
		}
		uv, ok := item.Field(this.plan.Alias())
		if !ok {
			context.Error(errors.NewUpdateAliasMissingError(this.plan.Alias()))
			return false
		}

		av, ok := uv.(value.AnnotatedValue)
		if !ok {
			context.Error(errors.NewUpdateAliasMetadataError(this.plan.Alias()))
			return false
		}

		key, ok := this.getDocumentKey(av, context)
		if !ok {
			return false
		}

		pairs = pairs[0 : i+1]
		pairs[i].Name = key

		var options value.Value
		var before uint64

		if context.UseRequestQuota() {
			before = item.Size()
		}
		clone := item.GetAttachment(value.ATT_CLONE)
		switch clone := clone.(type) {
		case value.AnnotatedValue:
			cv, ok := clone.Field(this.plan.Alias())
			if !ok {
				context.Error(errors.NewUpdateAliasMissingError(this.plan.Alias()))
				return false
			}

			cav := value.NewAnnotatedValue(cv)
			cav.CopyAnnotations(av)
			pairs[i].Value = cav

			if mv := clone.GetAttachment(value.ATT_OPTIONS); mv != nil {
				options, _ = mv.(value.Value)
			}

			// Adjust expiration to absolute value
			pairs[i].Options = adjustExpiration(options, false)
			// Update in the meta attachment so that it reflects in RETURNING clause
			setMetaExpiration(cav, pairs[i].Options, context.PreserveExpiry())

			item.SetField(this.plan.Alias(), cav)

		default:
			context.Error(errors.NewInvalidValueError(fmt.Sprintf(
				"Invalid UPDATE value of type %T.", clone)))
			return false
		}
		if context.UseRequestQuota() {
			after := item.RecalculateSize()
			if before > after {
				context.ReleaseValueSize(before - after)
			} else if before < after {
				err := context.TrackValueSize(after - before)
				if err != nil {
					context.Error(err)
					return false
				}
			}
		}
	}

	// If there is a RETURNING clause
	preserveMutations := !fastDiscard

	this.switchPhase(_SERVTIME)

	uCount, pairs, errs := this.keyspace.Update(pairs, &this.operatorCtx, preserveMutations)

	// Update mutation count with number of updated docs
	context.AddMutationCount(uint64(uCount))

	this.switchPhase(_EXECTIME)

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

func (this *SendUpdate) readonly() bool {
	return false
}

func (this *SendUpdate) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *SendUpdate) Done() {
	this.baseDone()
	if this.isComplete() {
		_SENDUPDATE_OP_POOL.Put(this)
	}
}

func getExpiration(options value.Value) (exptime uint32, present bool) {
	if options != nil && options.Type() == value.OBJECT {
		if v, ok := options.Field("expiration"); ok && v.Type() == value.NUMBER {
			present = true
			expiration := value.AsNumberValue(v).Int64()
			if expiration > 0 {
				exptime = uint32(expiration)
			}
		}
	}
	return
}

const _MONTH = uint32(30 * 24 * 60 * 60)

func adjustExpiration(options value.Value, copyOptions bool) value.Value {
	if options == nil || options.Type() != value.OBJECT {
		return options
	}
	expiration, present := getExpiration(options)
	if !present {
		return options
	}
	if expiration > 0 && expiration < _MONTH {
		expiration += uint32(time.Now().UTC().Unix())
	}
	if copyOptions {
		options = options.CopyForUpdate()
	}
	options.SetField("expiration", expiration)
	return options
}

func setMetaExpiration(av value.AnnotatedValue, options value.Value, preserveExpiry bool) {
	expiration, present := getExpiration(options)
	if !preserveExpiry || present {
		av.SetMetaField(value.META_EXPIRATION, expiration)
	}
}

var _UPDATE_POOL = value.NewPairPool(_BATCH_SIZE)
