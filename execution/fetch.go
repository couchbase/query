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

var _FETCH_OP_POOL util.FastPool
var _DUMMYFETCH_OP_POOL util.FastPool

const _MAX_RESULT_CACHE_SIZE = 1000

func init() {
	util.NewFastPool(&_FETCH_OP_POOL, func() interface{} {
		return &Fetch{}
	})
	util.NewFastPool(&_DUMMYFETCH_OP_POOL, func() interface{} {
		return &DummyFetch{}
	})
}

type Fetch struct {
	base
	plan       *plan.Fetch
	keyspace   datastore.Keyspace
	parentVal  value.Value
	deepCopy   bool
	hasCache   bool
	batchSize  int
	fetchCount uint64
	mk         missingKeys
	results    value.AnnotatedValues
	context    *Context
}

func NewFetch(plan *plan.Fetch, context *Context) *Fetch {
	rv := _FETCH_OP_POOL.Get().(*Fetch)
	rv.plan = plan
	rv.batchSize = context.GetPipelineBatch()
	rv.fetchCount = 0
	newBase(&rv.base, context)
	op := context.Type()
	rv.deepCopy = op == "" || op == "MERGE" || op == "UPDATE"
	rv.execPhase = FETCH
	rv.output = rv
	rv.mk.validate = plan.Term().ValidateKeys()
	rv.parentVal = nil

	return rv
}

func (this *Fetch) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFetch(this)
}

func (this *Fetch) Copy() Operator {
	rv := _FETCH_OP_POOL.Get().(*Fetch)
	rv.plan = this.plan
	rv.batchSize = this.batchSize
	rv.fetchCount = 0
	rv.deepCopy = this.deepCopy
	this.base.copy(&rv.base)
	rv.mk.validate = rv.plan.Term().ValidateKeys()
	return rv
}

func (this *Fetch) PlanOp() plan.Operator {
	return this.plan
}

func (this *Fetch) RunOnce(context *Context, parent value.Value) {
	if !this.plan.HasCacheResult() || !this.hasCache {
		this.runConsumer(this, context, parent, func() { this.releaseBatch(context) })
	} else {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		this.switchPhase(_EXECTIME)
		defer func() {
			this.notify()
			this.switchPhase(_NOTIME)
			this.close(context)
		}()

		if !active {
			return
		}

		// maxParallelism is 1. Use cached values
		alias := this.plan.Term().Alias()
		for _, sv := range this.results {
			key, hasKey := this.getDocumentKey(sv, context)
			if !hasKey {
				context.Error(errors.NewExecutionInternalError("Fetch: cached result value missing document key"))
				sv.Recycle()
				return
			}
			av := this.newEmptyDocumentWithKey(key, parent, context)
			var fv value.AnnotatedValue
			if this.deepCopy {
				fv = sv.CopyForUpdate().(value.AnnotatedValue)
			} else {
				fv = sv.Copy().(value.AnnotatedValue)
			}
			av.SetField(alias, fv)

			if context.UseRequestQuota() {
				err := context.TrackValueSize(av.Size())
				if err != nil {
					context.Error(err)
					av.Recycle()
					return
				}
			}

			if !this.sendItem(av) {
				av.Recycle()
				break
			}
		}
	}
}

func (this *Fetch) beforeItems(context *Context, parent value.Value) bool {
	this.mk.reset()
	if this.keyspace = this.plan.Keyspace(); this.keyspace == nil {
		this.keyspace = getKeyspace(this.keyspace, this.plan.Term().FromExpression(), &this.operatorCtx)
	}
	this.parentVal = parent
	if this.plan.HasCacheResult() && this.results == nil {
		this.results = make(value.AnnotatedValues, 0, _MAX_RESULT_CACHE_SIZE)
		this.context = context
	}
	return this.keyspace != nil
}

func (this *Fetch) processItem(item value.AnnotatedValue, context *Context) bool {
	item.ResetCovers(this.parentVal)
	ok := this.enbatchSize(item, this, this.batchSize, context, true)
	if ok {
		this.fetchCount++
		if this.fetchCount >= uint64(this.batchSize) {
			context.AddPhaseCount(FETCH, this.fetchCount)
			this.fetchCount = 0
		}
	}
	return ok
}

func (this *Fetch) afterItems(context *Context) {
	this.flushBatch(context)
	context.SetSortCount(0)
	context.AddPhaseCount(FETCH, this.fetchCount)
	this.fetchCount = 0
	this.releaseBatch(context)
	this.mk.report(context, this.plan.Term().Alias)
	// if this is the inner leg of a NL join, we don't want to repeatedly report the same keys as missing
	this.mk.validate = false
	this.parentVal = nil
	if this.plan.HasCacheResult() {
		this.hasCache = true
	}
}

func (this *Fetch) flushBatch(context *Context) bool {
	defer this.resetBatch(context)
	curQueue := this.queuedItems()
	if this.batchSize < curQueue {
		defer func() {
			size := int(this.output.ValueExchange().cap())
			if curQueue > size {
				curQueue = size
			}
			this.batchSize = curQueue
		}()
	}

	l := len(this.batch)
	if l == 0 || !this.isRunning() || this.stopped {
		return true
	}

	var keyCount map[string]int
	var fetchKeys []string

	fetchMap := _STRING_ANNOTATED_POOL.Get()
	defer _STRING_ANNOTATED_POOL.Put(fetchMap)

	cacheResult := this.plan.HasCacheResult()

	if l == 1 {
		var keys [1]string
		var ok bool

		keys[0], ok = this.getDocumentKey(this.batch[0], context)
		if !ok {
			return false
		}
		fetchKeys = keys[0:1:1]
	} else {
		keyCount = _STRING_KEYCOUNT_POOL.Get()
		defer _STRING_KEYCOUNT_POOL.Put(keyCount)

		fetchKeys = _STRING_POOL.Get()
		defer _STRING_POOL.Put(fetchKeys)

		for _, av := range this.batch {
			key, ok := this.getDocumentKey(av, context)
			if !ok {
				return false
			}

			v, ok := keyCount[key]
			if !ok {
				fetchKeys = append(fetchKeys, key)
				v = 0
			}
			keyCount[key] = v + 1
		}
	}

	this.switchPhase(_SERVTIME)

	var errs errors.Errors
	projection := this.plan.EarlyProjection()
	useSubDoc := !context.IsFeatureEnabled(util.N1QL_FULL_GET)

	// Fetch
	errs = this.keyspace.Fetch(fetchKeys, fetchMap, &this.operatorCtx, this.plan.SubPaths(), projection, useSubDoc)

	this.switchPhase(_EXECTIME)

	fetchOk := true
	for _, err := range errs {
		context.Error(err)
		if err.IsFatal() {
			fetchOk = false
		}
	}

	if l == 1 {
		fv := fetchMap[fetchKeys[0]]
		av := this.batch[0]
		if fv != nil {

			fv.SetAttachment(value.ATT_SMETA, av.GetAttachment(value.ATT_SMETA))
			av.SetField(this.plan.Term().Alias(), fv)

			if context.UseRequestQuota() {
				err := context.TrackValueSize(av.RecalculateSize())
				if err != nil {
					context.Error(err)
					av.Recycle()
					return false
				}
			}

			if cacheResult && !this.hasCache {
				var sfv value.AnnotatedValue
				if this.deepCopy {
					sfv = value.NewAnnotatedValue(fv.CopyForUpdate())
				} else {
					sfv = value.NewAnnotatedValue(fv.Copy())
				}
				if context.UseRequestQuota() {
					err := context.TrackValueSize(sfv.Size())
					if err != nil {
						context.Error(err)
						sfv.Recycle()
						return false
					}
				}
				if len(this.results) >= _MAX_RESULT_CACHE_SIZE {
					if this.plan.IsUnderNL() {
						context.Error(errors.NewNLInnerPrimaryDocsExceeded(this.plan.Term().Alias(), _MAX_RESULT_CACHE_SIZE))
					} else {
						context.Error(errors.NewSubqueryNumDocsExceeded(this.plan.Term().Alias(), _MAX_RESULT_CACHE_SIZE))
					}
					sfv.Recycle()
					return false
				}
				this.results = append(this.results, sfv)
			}

			if !this.sendItem(av) {
				av.Recycle()
				return false
			}
		} else {
			this.mk.add(fetchKeys[0])
		}
		return fetchOk
	}

	// Preserve order of keys
	for _, av := range this.batch {
		key, ok := this.getDocumentKey(av, context)
		if !ok {
			return false
		}

		fv := fetchMap[key]
		if fv != nil {
			if keyCount[key] > 1 {
				if this.deepCopy {
					fv = value.NewAnnotatedValue(fv.CopyForUpdate())
				} else {
					fv = value.NewAnnotatedValue(fv.Copy())
				}
			}
			keyCount[key]--

			if sm := av.GetAttachment(value.ATT_SMETA); sm != nil {
				fv.SetAttachment(value.ATT_SMETA, sm)
			}
			av.SetField(this.plan.Term().Alias(), fv)

			if context.UseRequestQuota() {
				err := context.TrackValueSize(av.RecalculateSize())
				if err != nil {
					context.Error(err)
					av.Recycle()
					return false
				}
			}

			if cacheResult && !this.hasCache {
				var sfv value.AnnotatedValue
				if this.deepCopy {
					sfv = value.NewAnnotatedValue(fv.CopyForUpdate())
				} else {
					sfv = value.NewAnnotatedValue(fv.Copy())
				}
				if context.UseRequestQuota() {
					err := context.TrackValueSize(sfv.Size())
					if err != nil {
						context.Error(err)
						sfv.Recycle()
						return false
					}
				}
				if len(this.results) >= _MAX_RESULT_CACHE_SIZE {
					if this.plan.IsUnderNL() {
						context.Error(errors.NewNLInnerPrimaryDocsExceeded(this.plan.Term().Alias(), _MAX_RESULT_CACHE_SIZE))
					} else {
						context.Error(errors.NewSubqueryNumDocsExceeded(this.plan.Term().Alias(), _MAX_RESULT_CACHE_SIZE))
					}
					sfv.Recycle()
					return false
				}
				this.results = append(this.results, sfv)
			}

			if !this.sendItem(av) {
				av.Recycle()
				return false
			}
		} else {
			this.mk.add(key)
		}
	}

	return fetchOk
}

func (this *Fetch) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *Fetch) Done() {
	this.baseDone()
	if this.plan.HasCacheResult() && this.results != nil {
		context := this.context
		for _, av := range this.results {
			if context != nil && context.UseRequestQuota() {
				context.ReleaseValueSize(av.Size())
			}
			av.Recycle()
		}
		this.results = nil
		this.hasCache = false
	}
	if this.isComplete() {
		_FETCH_OP_POOL.Put(this)
	}
}

type DummyFetch struct {
	base
	plan *plan.DummyFetch
}

func NewDummyFetch(plan *plan.DummyFetch, context *Context) *DummyFetch {
	rv := _DUMMYFETCH_OP_POOL.Get().(*DummyFetch)
	rv.plan = plan
	newBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *DummyFetch) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDummyFetch(this)
}

func (this *DummyFetch) Copy() Operator {
	rv := _DUMMYFETCH_OP_POOL.Get().(*DummyFetch)
	rv.plan = this.plan
	this.base.copy(&rv.base)
	return rv
}

func (this *DummyFetch) PlanOp() plan.Operator {
	return this.plan
}

func (this *DummyFetch) RunOnce(context *Context, parent value.Value) {
	this.runConsumer(this, context, parent, nil)
}

func (this *DummyFetch) processItem(item value.AnnotatedValue, context *Context) bool {
	item.SetField(this.plan.Term().Alias(), item.Copy())
	return this.sendItem(item)
}

func (this *DummyFetch) afterItems(context *Context) {
	context.SetSortCount(0)
}

func (this *DummyFetch) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}

func (this *DummyFetch) Done() {
	this.baseDone()
	if this.isComplete() {
		_DUMMYFETCH_OP_POOL.Put(this)
	}
}
