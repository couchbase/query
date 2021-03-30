//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package util

type InterfacePool struct {
	pool FastPool
	size int
}

func NewInterfacePool(size int) *InterfacePool {
	rv := &InterfacePool{
		size: size,
	}
	NewFastPool(&rv.pool, func() interface{} {
		return make([]interface{}, 0, size)
	})

	return rv
}

func (this *InterfacePool) Get() []interface{} {
	return this.pool.Get().([]interface{})
}

func (this *InterfacePool) GetCapped(capacity int) []interface{} {
	if capacity > this.size {
		return make([]interface{}, 0, capacity)
	} else {
		return this.Get()
	}
}

func (this *InterfacePool) GetSized(length int) []interface{} {
	if length > this.size {
		return make([]interface{}, length)
	}

	rv := this.Get()
	rv = rv[0:length]
	for i := 0; i < length; i++ {
		rv[i] = nil
	}

	return rv
}

func (this *InterfacePool) Put(s []interface{}) {
	if cap(s) != this.size {
		return
	}

	for i := range s {
		s[i] = nil
	}
	this.pool.Put(s[0:0])
}
