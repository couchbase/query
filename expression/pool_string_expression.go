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

type StringExpressionPool struct {
	pool util.FastPool
	size int
}

func NewStringExpressionPool(size int) *StringExpressionPool {
	rv := &StringExpressionPool{
		size: size,
	}
	util.NewFastPool(&rv.pool, func() interface{} {
		return make(map[string]Expression, size)
	})

	return rv
}

func (this *StringExpressionPool) Get() map[string]Expression {
	return this.pool.Get().(map[string]Expression)
}

func (this *StringExpressionPool) Put(s map[string]Expression) {
	if s == nil || len(s) > this.size {
		return
	}

	for k, _ := range s {
		s[k] = nil
		delete(s, k)
	}

	this.pool.Put(s)
}
