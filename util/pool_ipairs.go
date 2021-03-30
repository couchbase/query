//  Copyright 2016-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package util

type IPairsPool struct {
	pool FastPool
	size int
}

func NewIPairsPool(size int) *IPairsPool {
	rv := &IPairsPool{
		size: size,
	}
	NewFastPool(&rv.pool, func() interface{} {
		return make([][]IPair, 0, size)
	})

	return rv
}

func (this *IPairsPool) Get() [][]IPair {
	return this.pool.Get().([][]IPair)
}

func (this *IPairsPool) GetSized(length int) [][]IPair {
	if length > this.size {
		return make([][]IPair, length)
	}

	rv := this.Get()
	rv = rv[0:length]
	return rv
}

func (this *IPairsPool) Put(s [][]IPair) {
	if cap(s) != this.size {
		return
	}

	for i := range s {
		s[i] = nil
	}
	this.pool.Put(s[0:0])
}
