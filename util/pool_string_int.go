//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package util

type StringIntPool struct {
	pool FastPool
	size int
}

func NewStringIntPool(size int) *StringIntPool {
	rv := &StringIntPool{
		size: size,
	}
	NewFastPool(&rv.pool, func() interface{} {
		return make(map[string]int, rv.size)
	})

	return rv
}

func (this *StringIntPool) Get() map[string]int {
	return this.pool.Get().(map[string]int)
}

func (this *StringIntPool) Put(s map[string]int) {
	if s == nil || len(s) > this.size {
		return
	}

	for k, _ := range s {
		delete(s, k)
	}

	this.pool.Put(s)
}
