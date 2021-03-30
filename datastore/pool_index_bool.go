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

type IndexBoolPool struct {
	pool util.FastPool
	size int
}

func NewIndexBoolPool(size int) *IndexBoolPool {
	rv := &IndexBoolPool{
		size: size,
	}
	util.NewFastPool(&rv.pool, func() interface{} {
		return make(map[Index]bool, size)
	})

	return rv
}

func (this *IndexBoolPool) Get() map[Index]bool {
	return this.pool.Get().(map[Index]bool)
}

func (this *IndexBoolPool) Put(s map[Index]bool) {
	if s == nil || len(s) > this.size {
		return
	}

	for k, _ := range s {
		delete(s, k)
	}

	this.pool.Put(s)
}
