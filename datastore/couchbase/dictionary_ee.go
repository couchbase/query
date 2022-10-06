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

func dropDictCacheEntry(keyspace string) {
	dictionary.DropKeyspace(keyspace)
}

const _GRACE_PERIOD = time.Second

type chkIndexDict struct {
	sync.RWMutex
	checking  bool
	lastCheck time.Time
}

func (this *chkIndexDict) chkIndex() bool {
	if time.Since(this.lastCheck) <= _GRACE_PERIOD {
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
	this.lastCheck = time.Now()
	this.Unlock()
}

func (this *keyspace) checkIndexCache(indexer datastore.Indexer) errors.Error {
	if this.chkIndex == nil {
		this.chkIndex = &chkIndexDict{}
	}
	if !this.chkIndex.chkIndex() {
		return nil
	}

	defer this.chkIndex.chkDone()

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

		indexes[idx.Name()] = true
	}

	dictionary.CheckIndexes(this.Id(), indexes)

	return nil
}

var _INDEX_ID_POOL = util.NewStringBoolPool(256)
