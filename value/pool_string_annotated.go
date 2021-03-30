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

type StringAnnotatedPool struct {
	pool util.FastPool
	size int
}

func NewStringAnnotatedPool(size int) *StringAnnotatedPool {
	rv := &StringAnnotatedPool{
		size: size,
	}
	util.NewFastPool(&rv.pool, func() interface{} {
		return make(map[string]AnnotatedValue, rv.size)
	})

	return rv
}

func (this *StringAnnotatedPool) Get() map[string]AnnotatedValue {
	return this.pool.Get().(map[string]AnnotatedValue)
}

func (this *StringAnnotatedPool) Put(s map[string]AnnotatedValue) {
	if s == nil || len(s) > this.size {
		return
	}

	for k, _ := range s {
		s[k] = nil
		delete(s, k)
	}

	this.pool.Put(s)
}

func (this *StringAnnotatedPool) Size() int {
	return this.size
}
