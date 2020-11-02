// Copyright (c) 2019 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you
// may not use this file except in compliance with the License. You
// may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.
//
// The enterprise edition has access to couchbase/query-ee, which
// includes dictionary cache. This file is only built in with
// the enterprise edition.

// +build enterprise

package couchbase

import (
	"sync"
	"time"

	"github.com/couchbase/query-ee/dictionary"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/util"
)

type DictCacheEntry interface {
	Target(map[string]interface{})
	Dictionary(map[string]interface{})
}

func CountDictCacheEntries() int {
	return dictionary.CountDictCacheEntries()
}

func DictCacheEntriesForeach(nB func(string, interface{}) bool, b func() bool) {
	dictionary.DictCacheEntriesForeach(nB, b)
}

func DictCacheEntryDo(k string, f func(interface{})) {
	dictionary.DictCacheEntryDo(k, f)
}

func DropDictCacheEntry(keyspace string, remote bool) {
	dictionary.DropDictCacheEntry(keyspace, remote)
}

func NameDictCacheEntries() []string {
	return dictionary.NameDictCacheEntries()
}

// dictionary entries

func Get(key string) (DictCacheEntry, error) {
	ce, err := dictionary.Get(key)
	if err != nil {
		return nil, err
	}
	return ce.(DictCacheEntry), nil
}

func Count() (int64, error) {
	return dictionary.Count()
}

func Foreach(f func(string) error) error {
	return dictionary.Foreach(f)
}

func DropDictionaryEntry(keyspace string) {
	dictionary.DropDictionaryEntry(keyspace)
}

func DropDictEntryAndAllCache(keyspace string, context interface{}) {
	dictionary.DropDictEntryAndAllCache(keyspace, context)
}

const _GRACE_PERIOD = time.Second

type chkIndexDict struct {
	sync.RWMutex
	checking  bool
	lastCheck util.Time
}

func (this *chkIndexDict) chkIndex() bool {
	if util.Since(this.lastCheck) <= _GRACE_PERIOD {
		return false
	}

	this.RLock()
	if this.checking {
		this.RUnlock()
		return false
	}
	this.RUnlock()

	this.Lock()
	this.checking = true
	this.Unlock()
	return true
}

func (this *chkIndexDict) chkDone() {
	this.Lock()
	this.checking = false
	this.lastCheck = util.Now()
	this.Unlock()
}

func checkIndexCache(keyspace string, indexer datastore.Indexer, dict *chkIndexDict) errors.Error {
	if !dict.chkIndex() {
		return nil
	}

	defer dict.chkDone()

	indexes := _INDEX_ID_POOL.Get()
	defer _INDEX_ID_POOL.Put(indexes)

	idxes, err := indexer.Indexes()
	if err != nil {
		return err
	}

	for _, idx := range idxes {
		state, _, err := idx.State()
		if err != nil {
			return err
		}
		if state != datastore.ONLINE {
			continue
		}

		indexes[idx.Id()] = idx.Name()
	}

	dictionary.CheckIndexes(keyspace, indexes)

	return nil
}

var _INDEX_ID_POOL = util.NewStringStringPool(256)
