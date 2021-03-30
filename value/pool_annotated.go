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

type AnnotatedPool struct {
	pool util.FastPool
	size int
}

func NewAnnotatedPool(size int) *AnnotatedPool {
	rv := &AnnotatedPool{
		size: size,
	}
	util.NewFastPool(&rv.pool, func() interface{} {
		return make(AnnotatedValues, 0, size)
	})
	return rv
}

func (this *AnnotatedPool) Get() AnnotatedValues {
	return this.pool.Get().(AnnotatedValues)
}

func (this *AnnotatedPool) Put(s AnnotatedValues) {
	if cap(s) != this.size {
		return
	}

	for i := range s {
		s[i] = nil
	}
	this.pool.Put(s[0:0])
}

func (this *AnnotatedPool) Size() int {
	return this.size
}
