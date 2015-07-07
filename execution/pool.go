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
	"sync"

	"github.com/couchbase/query/datastore"
)

var _STRING_POOL = &sync.Pool{
	New: func() interface{} {
		return make([]string, 0, _BATCH_SIZE)
	},
}

func allocateStringBatch() []string {
	return _STRING_POOL.Get().([]string)
}

func releaseStringBatch(s []string) {
	if cap(s) != _BATCH_SIZE {
		return
	}

	_STRING_POOL.Put(s[0:0])
}

var _PAIR_POOL = &sync.Pool{
	New: func() interface{} {
		return make([]datastore.Pair, 0, _BATCH_SIZE)
	},
}

func allocatePairBatch() []datastore.Pair {
	return _PAIR_POOL.Get().([]datastore.Pair)
}

func releasePairBatch(p []datastore.Pair) {
	if cap(p) != _BATCH_SIZE {
		return
	}

	_PAIR_POOL.Put(p[0:0])
}
