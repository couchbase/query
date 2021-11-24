//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package datastore

import (
	"github.com/couchbase/query/util"
)

type IndexPool struct {
	pool util.FastPool
	size int
}

func NewIndexPool(size int) *IndexPool {
	rv := &IndexPool{
		size: size,
	}
	util.NewFastPool(&rv.pool, func() interface{} {
		return make([]Index, 0, size)
	})

	return rv
}

func (this *IndexPool) Get() []Index {
	return this.pool.Get().([]Index)
}

func (this *IndexPool) Put(s []Index) {
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
