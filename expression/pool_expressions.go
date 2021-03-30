//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package expression

import (
	"github.com/couchbase/query/util"
)

type ExpressionsPool struct {
	pool util.FastPool
	size int
}

func NewExpressionsPool(size int) *ExpressionsPool {
	rv := &ExpressionsPool{
		size: size,
	}
	util.NewFastPool(&rv.pool, func() interface{} {
		return make([]Expressions, 0, size)
	})

	return rv
}

func (this *ExpressionsPool) Get() []Expressions {
	return this.pool.Get().([]Expressions)
}

func (this *ExpressionsPool) GetSized(length int) []Expressions {
	if length > this.size {
		return make([]Expressions, length)
	}

	rv := this.Get()
	rv = rv[0:length]
	for i := 0; i < length; i++ {
		rv[i] = nil
	}

	return rv
}

func (this *ExpressionsPool) Put(s []Expressions) {
	if cap(s) != this.size {
		return
	}

	this.pool.Put(s[0:0])
}

func (this *ExpressionsPool) Size() int {
	return this.size
}
