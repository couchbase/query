// Copyright 2019-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
// file, in accordance with the Business Source License, use of this software
// will be governed by the Apache License, Version 2.0, included in the file
// licenses/APL2.txt.
//
// Currently, the community edition does not have access to dictionary
// cache, so this function is no-op.

//go:build !enterprise
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
