//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type joinBase struct {
	base
	joinBatch    []value.AnnotatedJoinPair
	joinKeyCount int
	mk           missingKeys
}

func newJoinBase(joinBase *joinBase, context *Context) {
	newBase(&joinBase.base, context)
}

func (this *joinBase) copy(joinBase *joinBase) {
	this.base.copy(&joinBase.base)
}

func (this *joinBase) allocateBatch(context *Context, size int) {
	if size <= PipelineBatchSize() {
		this.joinBatch = getJoinBatchPool().Get()
	} else {
		this.joinBatch = make(value.AnnotatedJoinPairs, 0, size)
	}
}

func (this *joinBase) releaseBatch(context *Context) {
	getJoinBatchPool().Put(this.joinBatch)
	this.joinBatch = nil
	this.joinKeyCount = 0
}

func (this *joinBase) resetBatch(context *Context) {
	if this.joinBatch != nil {
		for i := range this.joinBatch {
			this.joinBatch[i] = value.AnnotatedJoinPair{}
		}
		this.joinBatch = this.joinBatch[:0]
	}
	this.joinKeyCount = 0
}

func (this *joinBase) joinEnbatch(item value.AnnotatedJoinPair, b batcher, context *Context) bool {
	if (len(item.Keys)+this.joinKeyCount > cap(this.joinBatch)) || len(this.joinBatch) >= cap(this.joinBatch) {
		if !b.flushBatch(context) {
			return false
		}
	}

	if this.joinBatch == nil {
		this.allocateBatch(context, cap(this.joinBatch))
	}

	this.joinBatch = append(this.joinBatch, item)
	this.joinKeyCount += len(item.Keys)
	return true
}

func (this *joinBase) joinFetch(keyspace datastore.Keyspace, subPaths []string, projection []string, keyCount map[string]int,
	pairMap map[string]value.AnnotatedValue, context *Context) bool {

	fetchKeys := _STRING_POOL.Get()
	defer _STRING_POOL.Put(fetchKeys)

	for _, item := range this.joinBatch {
		for _, key := range item.Keys {
			v, ok := keyCount[key]
			if !ok {
				fetchKeys = append(fetchKeys, key)
				v = 0
			}
			keyCount[key] = v + 1
		}
	}

	this.switchPhase(_SERVTIME)
	errs := keyspace.Fetch(fetchKeys, pairMap, context, subPaths, projection, !context.IsFeatureEnabled(util.N1QL_FULL_GET))
	this.switchPhase(_EXECTIME)

	fetchOk := true
	for _, err := range errs {
		context.Error(err)
		if err.IsFatal() {
			fetchOk = false
		}
	}

	if context.UseRequestQuota() {
		var size uint64

		for _, val := range pairMap {
			size += val.Size()
		}
		err := context.TrackValueSize(size)
		if err != nil {
			context.Error(err)
			fetchOk = false
		}
	}

	return fetchOk
}

func (this *joinBase) joinEntries(keyCount map[string]int, pairMap map[string]value.AnnotatedValue,
	outer bool, onFilter expression.Expression, alias string, context *opContext) bool {

	useQuota := context.UseRequestQuota()
	for _, item := range this.joinBatch {
		foundKeys := 0
		if len(pairMap) > 0 {
			for _, key := range item.Keys {
				_, ok := pairMap[key]
				if ok {
					foundKeys++
				}
			}
		}

		matched := false
		if foundKeys != 0 {
			for _, key := range item.Keys {
				var alreadyAccountedForSize uint64

				pv, ok := pairMap[key]
				if !ok {
					continue
				}

				var joined value.AnnotatedValue
				if foundKeys > 1 || (outer && onFilter != nil) {
					joined = value.NewAnnotatedValue(item.Value.Copy())
				} else {
					joined = item.Value
					if useQuota {
						alreadyAccountedForSize += joined.Size()
					}
				}
				foundKeys--

				var av value.AnnotatedValue
				if keyCount[key] > 1 {
					av = value.NewAnnotatedValue(pv.Copy())
				} else {
					av = pv
					if useQuota {
						alreadyAccountedForSize += av.Size()
					}
				}
				keyCount[key]--

				joined.SetField(alias, av)

				if onFilter != nil {
					result, err := onFilter.Evaluate(joined, context)
					if err != nil {
						context.Error(errors.NewEvaluationError(err, "lookup join filter"))
						return false
					}
					if !result.Truth() {
						continue
					}
				}

				if useQuota {
					err := context.TrackValueSize(joined.RecalculateSize() - alreadyAccountedForSize)
					if err != nil {
						context.Error(err)
						return false
					}
				}

				matched = true
				if !this.sendItem(joined) {
					return false
				}
			}
		}
		if outer && !matched {
			if !this.sendItem(item.Value) {
				return false
			}
		} else if useQuota {
			context.ReleaseValueSize(item.Value.Size())
		}
	}

	return true
}

func (this *joinBase) nestEntries(keyCount map[string]int, pairMap map[string]value.AnnotatedValue,
	outer bool, onFilter expression.Expression, alias string, context *opContext) bool {

	useQuota := context.UseRequestQuota()
	for _, item := range this.joinBatch {
		var alreadyAccountedForSize uint64

		av := item.Value
		nvs := make([]interface{}, 0, len(item.Keys))
		if len(pairMap) > 0 {
			for _, key := range item.Keys {
				pv, ok := pairMap[key]
				if !ok {
					continue
				}

				var jv value.AnnotatedValue
				if keyCount[key] > 1 {
					jv = value.NewAnnotatedValue(pv.Copy())
				} else {
					jv = pv
					if useQuota {
						alreadyAccountedForSize += jv.Size()
					}
				}
				keyCount[key]--

				if onFilter != nil {
					result, err := onFilter.Evaluate(jv, context)
					if err != nil {
						context.Error(errors.NewEvaluationError(err, "lookup nest filter"))
						return false
					}
					if !result.Truth() {
						continue
					}
				}
				nvs = append(nvs, jv)
			}
		}

		if len(nvs) != 0 {
			if useQuota {
				alreadyAccountedForSize += av.Size()
				av.SetField(alias, nvs)
				err := context.TrackValueSize(av.RecalculateSize() - alreadyAccountedForSize)
				if err != nil {
					context.Error(err)
					av.Recycle()
					return false
				}
			} else {
				av.SetField(alias, nvs)
			}
			if !this.sendItem(av) {
				return false
			}
		} else if outer {
			if len(item.Keys) != 0 {
				// non missing keys
				if useQuota {
					alreadyAccountedForSize = av.Size()
					av.SetField(alias, value.EMPTY_ARRAY_VALUE)
					err := context.TrackValueSize(av.RecalculateSize() - alreadyAccountedForSize)
					if err != nil {
						context.Error(err)
						av.Recycle()
						return false
					}
				} else {
					av.SetField(alias, value.EMPTY_ARRAY_VALUE)
				}
			}
			if !this.sendItem(av) {
				return false
			}
		} else if useQuota {
			context.ReleaseValueSize(item.Value.Size())
		}
	}

	return true
}

func (this *joinBase) reopen(context *Context) bool {
	rv := this.baseReopen(context)
	if this.batch != nil {
		this.releaseBatch(context)
	}
	return rv
}

func (this *joinBase) validateKeys(pairMap map[string]value.AnnotatedValue) {
	if !this.mk.validate {
		return
	}
	for _, item := range this.joinBatch {
		for _, key := range item.Keys {
			if _, ok := pairMap[key]; !ok {
				this.mk.add(key)
			}
		}
	}
}

var _STRING_KEYCOUNT_POOL = util.NewStringIntPool(_MAP_POOL_CAP)
