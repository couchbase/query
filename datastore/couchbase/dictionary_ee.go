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

package couchbase

import (
	"sync"
	"time"

	"github.com/couchbase/query-ee/dictionary"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	functionStorage "github.com/couchbase/query/functions/storage"
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

func Count(bucketName string, context datastore.QueryContext, check func(context datastore.QueryContext, ds datastore.Datastore,
	elems ...string) bool) (int64, error) {

	if isSysBucket(bucketName) {
		return 0, nil
	}
	return dictionary.Count(bucketName, context, check)
}

func Foreach(bucketName string, context datastore.QueryContext, check func(context datastore.QueryContext, ds datastore.Datastore,
	elems ...string) bool, proc func(string) error) error {

	if isSysBucket(bucketName) {
		return nil
	}
	return dictionary.Foreach(bucketName, context, check, proc)
}

func DropDictionaryEntry(keyspace string, isDropBucket bool, locked bool) {
	if isSysBucket(keyspace) || dictionary.IsSysCBOStats(keyspace) {
		dictionary.DropDictionaryCache()
	} else {
		sysStore := dictionary.UseSystemStorage()
		if sysStore && isDropBucket {
			// if using _system scope, if bucket is being dropped, only need to drop
			// from dictionary cache; the _system scope is being dropped as part
			// of bucket drop and thus no need to remove entries from there
			dictionary.DropDictCacheEntry(keyspace, false)
		} else {
			dictionary.DropDictionaryEntry(keyspace, sysStore, locked)
		}
	}
}

func DropDictEntryAndAllCache(keyspace string, context interface{}, locked bool) {
	if isSysBucket(keyspace) || dictionary.IsSysCBOStats(keyspace) {
		dictionary.DropDictionaryCache()
	} else {
		dictionary.DropDictEntryAndAllCache(keyspace, dictionary.UseSystemStorage(), context, locked)
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

func GetCBOKeyspace(key string) (string, bool) {
	return dictionary.GetKeyspace(key)
}

const _GRACE_PERIOD = 2 * time.Second

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

	return dictionary.CheckIndexes(keyspace, indexer)
}

func SupportedBackupVersion() int {
	v1 := functionStorage.SupportedBackupVersion()
	if v1 == datastore.BACKUP_NOT_POSSIBLE {
		return v1
	}
	v2 := dictionary.SupportedBackupVersion()
	if v2 == datastore.BACKUP_NOT_POSSIBLE {
		return v2
	}
	if v1 == datastore.CURRENT_BACKUP_VERSION && v2 == datastore.CURRENT_BACKUP_VERSION {
		return datastore.CURRENT_BACKUP_VERSION
	} else if v1 == datastore.CURRENT_BACKUP_VERSION || v2 == datastore.CURRENT_BACKUP_VERSION {
		// if one is reporting a specific version and the other CURRENT, then there was a migration issue and we should not
		// allow backups to proceed
		return datastore.BACKUP_NOT_POSSIBLE
	} else if v1 < v2 {
		return v1
	}
	return v2
}
