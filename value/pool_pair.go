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

type PairPool struct {
	pool util.FastPool
	size int
}

func NewPairPool(size int) *PairPool {
	rv := &PairPool{
		size: size,
	}
	util.NewFastPool(&rv.pool, func() interface{} {
		return make([]Pair, 0, size)
	})

	return rv
}

func (this *PairPool) Get() []Pair {
	return this.pool.Get().([]Pair)
}

func (this *PairPool) Put(s []Pair) {
	if cap(s) != this.size {
		return
	}

	for i := range s {
		s[i] = Pair{}
	}
	this.pool.Put(s[0:0])
}

func (this *PairPool) Size() int {
	return this.size
}
