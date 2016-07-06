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
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type joinBase struct {
	base
	joinBatch    []value.AnnotatedJoinPair
	joinKeyCount int
}

func newJoinBase() joinBase {
	return joinBase{
		base: newBase(),
	}
}

func (this *joinBase) copy() joinBase {
	return joinBase{
		base: this.base.copy(),
	}
}

func (this *joinBase) allocateBatch() {
	this.joinBatch = getJoinBatchPool().Get()
}

func (this *joinBase) releaseBatch() {
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
		this.allocateBatch()
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

	pairs, errs := keyspace.Fetch(fetchKeys)

	fetchOk := true
	for _, err := range errs {
		context.Error(err)
		if err.IsFatal() {
			fetchOk = false
		}
	}

	if fetchOk {
		for _, pair := range pairs {
			pairMap[pair.Name] = pair.Value
		}
	}

	return fetchOk
}

func (this *joinBase) joinEntries(keyCount map[string]int, pairMap map[string]value.AnnotatedValue, outer bool, alias string) bool {
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

		if foundKeys != 0 {
			for _, key := range item.Keys {
				pv, ok := pairMap[key]
				if !ok {
					continue
				}

				var joined value.AnnotatedValue
				if foundKeys > 1 {
					joined = value.NewAnnotatedValue(item.Value.Copy())
				} else {
					joined = item.Value
				}
				foundKeys--

				var av value.AnnotatedValue
				if keyCount[key] > 1 {
					av = value.NewAnnotatedValue(pv.Copy())
				} else {
					av = pv
				}
				keyCount[key]--

				joined.SetField(alias, av)

				if !this.sendItem(joined) {
					return false
				}
			}
		} else if outer && !this.sendItem(item.Value) {
			return false
		}
	}

	return true
}

func (this *joinBase) nestEntries(keyCount map[string]int, pairMap map[string]value.AnnotatedValue,
	outer bool, alias string) bool {
	for _, item := range this.joinBatch {
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
				}
				keyCount[key]--

				nvs = append(nvs, jv)
			}
		}

		if len(nvs) != 0 {
			av.SetField(alias, nvs)
			if !this.sendItem(av) {
				return false
			}
		} else if outer {
			av.SetField(alias, value.EMPTY_ARRAY_VALUE)
			if !this.sendItem(av) {
				return false
			}
		}
	}

	return true
}

var _STRING_KEYCOUNT_POOL = util.NewStringIntPool(_MAP_POOL_CAP)
