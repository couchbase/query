//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.
//
//go:build enterprise

package server

import (
	"github.com/couchbase/query-ee/dictionary"
)

func InitDictionaryCache(dictCacheLimit int) {
	dictionary.DictionaryCacheInit(dictCacheLimit)
}

func MigrateDictionary() {
	dictionary.Migrate()
}
