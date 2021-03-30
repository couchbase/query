//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package util

type BoolPool struct {
	pool FastPool
	size int
}

func NewBoolPool(size int) *BoolPool {
	rv := &BoolPool{
		size: size,
	}
	NewFastPool(&rv.pool, func() interface{} {
		return make([]bool, 0, size)
	})

	return rv
}

func (this *BoolPool) Get() []bool {
	return this.pool.Get().([]bool)
}

func (this *BoolPool) GetCapped(capacity int) []bool {
	if capacity > this.size {
		return make([]bool, 0, capacity)
	} else {
		return this.Get()
	}
}

func (this *BoolPool) GetSized(length int) []bool {
	if length > this.size {
		return make([]bool, length)
	}

	rv := this.Get()
	rv = rv[0:length]
	for i := 0; i < length; i++ {
		rv[i] = false
	}

	return rv
}

func (this *BoolPool) Put(s []bool) {
	if cap(s) != this.size {
		return
	}

	this.pool.Put(s[0:0])
}
