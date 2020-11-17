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

func (this *joinBase) joinFetch(keyspace datastore.Keyspace, keyCount map[string]int,
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
	errs := keyspace.Fetch(fetchKeys, pairMap, context, nil)
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
		if context.TrackValueSize(size) {
			context.Error(errors.NewMemoryQuotaExceededError())
			fetchOk = false
		}
	}

	return fetchOk
}

func (this *joinBase) joinEntries(keyCount map[string]int, pairMap map[string]value.AnnotatedValue,
	outer bool, onFilter expression.Expression, alias string, context *Context) bool {
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
			useQuota := context.UseRequestQuota()
			for _, key := range item.Keys {
				var size uint64

				pv, ok := pairMap[key]
				if !ok {
					continue
				}

				var joined value.AnnotatedValue
				if foundKeys > 1 || (outer && onFilter != nil) {
					joined = value.NewAnnotatedValue(item.Value.Copy())
					if useQuota {
						size += joined.Size()
					}
				} else {
					joined = item.Value
				}
				foundKeys--

				var av value.AnnotatedValue
				if keyCount[key] > 1 {
					av = value.NewAnnotatedValue(pv.Copy())
					if useQuota {
						size += av.Size()
					}
				} else {
					av = pv
				}
				keyCount[key]--

				joined.SetField(alias, av)

				if useQuota && context.TrackValueSize(size) {
					context.Error(errors.NewMemoryQuotaExceededError())
					return false
				}

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
		}
	}

	return true
}

func (this *joinBase) nestEntries(keyCount map[string]int, pairMap map[string]value.AnnotatedValue,
	outer bool, onFilter expression.Expression, alias string, context *Context) bool {
	useQuota := context.UseRequestQuota()

	for _, item := range this.joinBatch {
		var size uint64

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
					if useQuota {
						size += jv.Size()
					}
				} else {
					jv = pv
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
			av.SetField(alias, nvs)

			if useQuota && context.TrackValueSize(size) {
				context.Error(errors.NewMemoryQuotaExceededError())
				av.Recycle()
				return false
			}
			if !this.sendItem(av) {
				return false
			}
		} else if outer {
			if len(item.Keys) != 0 {
				// non missing keys
				av.SetField(alias, value.EMPTY_ARRAY_VALUE)
			}
			if !this.sendItem(av) {
				return false
			}
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

var _STRING_KEYCOUNT_POOL = util.NewStringIntPool(_MAP_POOL_CAP)
