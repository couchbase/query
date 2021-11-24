//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package util

type StringStringPool struct {
	pool FastPool
	size int
}

func NewStringStringPool(size int) *StringStringPool {
	rv := &StringStringPool{
		size: size,
	}
	NewFastPool(&rv.pool, func() interface{} {
		return make(map[string]string, rv.size)
	})

	return rv
}

func (this *StringStringPool) Get() map[string]string {
	return this.pool.Get().(map[string]string)
}

func (this *StringStringPool) Put(s map[string]string) {
	if s == nil || len(s) > this.size {
		return
	}

	for k, _ := range s {
		s[k] = ""
		delete(s, k)
	}

	this.pool.Put(s)
}
