//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package algebra

import (
	"github.com/couchbase/query/util"
)

type UnnestPool struct {
	pool util.FastPool
	size int
}

func NewUnnestPool(size int) *UnnestPool {
	rv := &UnnestPool{
		size: size,
	}
	util.NewFastPool(&rv.pool, func() interface{} {
		return make([]*Unnest, 0, size)
	})

	return rv
}

func (this *UnnestPool) Get() []*Unnest {
	return this.pool.Get().([]*Unnest)
}

func (this *UnnestPool) Put(buf []*Unnest) {
	if cap(buf) < this.size || cap(buf) > 2*this.size {
		return
	}

	for i := range buf {
		buf[i] = nil
	}
	this.pool.Put(buf[0:0])
}
