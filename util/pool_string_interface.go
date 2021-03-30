//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package util

type StringInterfacePool struct {
	pool FastPool
	size int
}

func NewStringInterfacePool(size int) *StringInterfacePool {
	rv := &StringInterfacePool{
		size: size,
	}
	NewFastPool(&rv.pool, func() interface{} {
		return make(map[string]interface{}, rv.size)
	})

	return rv
}

func (this *StringInterfacePool) Get() map[string]interface{} {
	return this.pool.Get().(map[string]interface{})
}

func (this *StringInterfacePool) GetCapped(capacity int) map[string]interface{} {
	if capacity <= this.size {
		return this.Get()
	} else {
		return make(map[string]interface{}, capacity)
	}
}

func (this *StringInterfacePool) Put(s map[string]interface{}) {
	if s == nil || len(s) > this.size {
		return
	}

	for k, _ := range s {
		s[k] = nil
		delete(s, k)
	}

	this.pool.Put(s)
}
