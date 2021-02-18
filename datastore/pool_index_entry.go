//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package datastore

import (
	"github.com/couchbase/query/util"
)

type IndexEntryPool struct {
	pool util.FastPool
	size int
}

func NewIndexEntryPool(size int) *IndexEntryPool {
	rv := &IndexEntryPool{
		size: size,
	}
	util.NewFastPool(&rv.pool, func() interface{} {
		return make([]*IndexEntry, 0, size)
	})

	return rv
}

func (this *IndexEntryPool) Get() []*IndexEntry {
	return this.pool.Get().([]*IndexEntry)
}

func (this *IndexEntryPool) Put(s []*IndexEntry) {
	if cap(s) != this.size {
		return
	}

	s = s[:cap(s)]
	for i := range s {
		if s[i] == nil {
			break
		}
		s[i] = nil
	}
	this.pool.Put(s[0:0])
}
