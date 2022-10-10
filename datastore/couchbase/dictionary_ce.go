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

func dropDictCacheEntry(keyspace string) {
	// no-op
}

type chkIndexDict struct {
	// dummy struct
}

func (this *keyspace) checkIndexCache(indexer datastore.Indexer) errors.Error {
	return nil
}
