//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package value

import (
	"github.com/couchbase/query/util"
)

type ValuePool struct {
	pool util.FastPool
	size int
}

func NewValuePool(size int) *ValuePool {
	rv := &ValuePool{
		size: size,
	}
	util.NewFastPool(&rv.pool, func() interface{} {
		return make([]Value, 0, size)
	})
	return rv
}

func (this *ValuePool) Get() []Value {
	return this.pool.Get().([]Value)
}

func (this *ValuePool) GetSized(length int) []Value {
	if length > this.size {
		return make([]Value, length)
	}

	rv := this.Get()
	rv = rv[0:length]
	for i := 0; i < length; i++ {
		rv[i] = nil
	}

	return rv
}

func (this *ValuePool) Put(s []Value) {
	if cap(s) != this.size {
		return
	}

	for i := range s {
		s[i] = nil
	}
	this.pool.Put(s[0:0])
}

func (this *ValuePool) Size() int {
	return this.size
}
