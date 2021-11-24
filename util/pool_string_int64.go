//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package util

type StringInt64Pool struct {
	pool FastPool
	size int
}

func NewStringInt64Pool(size int) *StringInt64Pool {
	rv := &StringInt64Pool{
		size: size,
	}
	NewFastPool(&rv.pool, func() interface{} {
		return make(map[string]int64, size)
	})

	return rv
}

func (this *StringInt64Pool) Get() map[string]int64 {
	return this.pool.Get().(map[string]int64)
}

func (this *StringInt64Pool) Put(s map[string]int64) {
	if s == nil || len(s) > this.size {
		return
	}

	for k, _ := range s {
		delete(s, k)
	}

	this.pool.Put(s)
}
