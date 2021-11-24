//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/util"
)

type SargSpansPool struct {
	pool util.FastPool
	size int
}

func NewSargSpansPool(size int) *SargSpansPool {
	rv := &SargSpansPool{
		size: size,
	}
	util.NewFastPool(&rv.pool, func() interface{} {
		return make([]SargSpans, 0, size)
	})

	return rv
}

func (this *SargSpansPool) Get() []SargSpans {
	return this.pool.Get().([]SargSpans)
}

func (this *SargSpansPool) GetSized(length int) []SargSpans {
	if length > this.size {
		return make([]SargSpans, length)
	}

	rv := this.Get()
	rv = rv[0:length]

	return rv
}

func (this *SargSpansPool) Put(s []SargSpans) {
	if cap(s) != this.size {
		return
	}

	for i := range s {
		s[i] = nil
	}
	this.pool.Put(s[0:0])
}

func (this *SargSpansPool) Size() int {
	return this.size
}
