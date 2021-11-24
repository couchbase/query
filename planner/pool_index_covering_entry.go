//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/util"
)

// fastpool for covered index

type IndexCoveringEntryPool struct {
	pool util.FastPool
	size int
}

func NewIndexCoveringEntryPool(size int) *IndexCoveringEntryPool {
	rv := &IndexCoveringEntryPool{
		size: size,
	}
	util.NewFastPool(&rv.pool, func() interface{} {
		return make(map[datastore.Index]*coveringEntry, size)
	})

	return rv
}

func (this *IndexCoveringEntryPool) Get() map[datastore.Index]*coveringEntry {
	return this.pool.Get().(map[datastore.Index]*coveringEntry)
}

func (this *IndexCoveringEntryPool) Put(s map[datastore.Index]*coveringEntry) {
	if s == nil || len(s) > this.size {
		return
	}

	for k, _ := range s {
		delete(s, k)
	}

	this.pool.Put(s)
}

var _COVERING_ENTRY_POOL = NewIndexCoveringEntryPool(16)
