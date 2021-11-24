//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import (
	"github.com/couchbase/query/util"
)

type StringSpanPool struct {
	pool util.FastPool
	size int
}

func NewStringSpanPool(size int) *StringSpanPool {
	rv := &StringSpanPool{
		size: size,
	}
	util.NewFastPool(&rv.pool, func() interface{} {
		return make(map[string]*Span2, size)
	})

	return rv
}

func (this *StringSpanPool) Get() map[string]*Span2 {
	return this.pool.Get().(map[string]*Span2)
}

func (this *StringSpanPool) Put(s map[string]*Span2) {
	if s == nil || len(s) > this.size {
		return
	}

	for k, _ := range s {
		s[k] = nil
		delete(s, k)
	}

	this.pool.Put(s)
}
