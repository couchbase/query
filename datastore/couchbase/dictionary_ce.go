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
// Currently, the community edition does not have access to dictionary
// cache, so this function is no-op.

// +build !enterprise

package couchbase

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
)

// dictionary cache entries

// dummy
type DictCacheEntry interface {
	Target(map[string]interface{})
	Dictionary(map[string]interface{})
}

func CountDictCacheEntries() int {
	return -1
}

func DictCacheEntriesForeach(nB func(string, interface{}) bool, b func() bool) {
	// no-op
}

func DictCacheEntryDo(k string, f func(interface{})) {
	// no-op
}

func DropDictCacheEntry(keyspace string, remote bool) {
	// no-op
}

func NameDictCacheEntries() []string {
	return []string{}
}

// dictionary entries

func Get(key string) (DictCacheEntry, error) {
	return nil, nil
}

func Count() (int64, error) {
	return -1, nil
}

func Foreach(f func(string) error) error {
	return nil
}

func DropDictionaryEntry(keyspace string) {
	// no-op
}

func DropDictEntryAndAllCache(keyspace string, context interface{}) {
	// no-op
}

func DropDictionaryCache() {
	// no-op
}

func isSysBucket(name string) bool {
	return false
}

func chkSysBucket() {
	// no-op
}

type chkIndexDict struct {
	// dummy struct
}

func checkIndexCache(keyspace string, indexer datastore.Indexer, dict *chkIndexDict) errors.Error {
	return nil
}
