// Copyright 2019-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
// file, in accordance with the Business Source License, use of this software
// will be governed by the Apache License, Version 2.0, included in the file
// licenses/APL2.txt.
//
// The enterprise edition has access to couchbase/query-ee, which
// includes dictionary cache. This file is only built in with
// the enterprise edition.

//go:build enterprise
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
	if isSysBucket(keyspace) || dictionary.IsSysCBOStats(keyspace) {
		dictionary.DropDictionaryCache()
	} else {
		dictionary.DropDictionaryEntry(keyspace)
	}
}

func DropDictEntryAndAllCache(keyspace string, context interface{}) {
	if isSysBucket(keyspace) || dictionary.IsSysCBOStats(keyspace) {
		dictionary.DropDictionaryCache()
	} else {
		dictionary.DropDictEntryAndAllCache(keyspace, context)
	}
}

func DropDictionaryCache() {
	dictionary.DropDictionaryCache()
}

func isSysBucket(name string) bool {
	return name == _N1QL_SYSTEM_BUCKET
}

func chkSysBucket() {
	// Bucket updater could be triggered by creation of N1QL_SYSTEM_SCOPE/N1QL_CBO_STATS
	// ignore these
	if dictionary.IsCreatingSysCBOStats() {
		return
	}

	hasSysCBOStats, err := dictionary.CheckSysCBOStats(false, "", true, true)
	if err == nil && !hasSysCBOStats {
		// N1QL_SYSTEM_SCOPE or N1QL_CBO_STATS is dropped
		dictionary.DropDictionaryCache()
	}
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
