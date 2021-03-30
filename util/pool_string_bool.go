//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package util

type StringBoolPool struct {
	pool FastPool
	size int
}

func NewStringBoolPool(size int) *StringBoolPool {
	rv := &StringBoolPool{
		size: size,
	}
	NewFastPool(&rv.pool, func() interface{} {
		return make(map[string]bool, size)
	})

	return rv
}

func (this *StringBoolPool) Get() map[string]bool {
	return this.pool.Get().(map[string]bool)
}

func (this *StringBoolPool) Put(s map[string]bool) {
	if s == nil || len(s) > this.size {
		return
	}

	for k, _ := range s {
		delete(s, k)
	}

	this.pool.Put(s)
}

func (this *StringBoolPool) Size() int {
	return this.size
}
