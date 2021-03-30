//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

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
