//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package execution

import (
	"github.com/couchbase/query/util"
)

type OperatorPool struct {
	pool util.FastPool
	size int
}

func NewOperatorPool(size int) *OperatorPool {
	rv := &OperatorPool{
		size: size,
	}
	util.NewFastPool(&rv.pool, func() interface{} {
		return make([]Operator, 0, size)
	})
	return rv
}

func (this *OperatorPool) Get() []Operator {
	return this.pool.Get().([]Operator)
}

func (this *OperatorPool) Put(s []Operator) {
	if cap(s) != this.size {
		return
	}

	this.pool.Put(s[0:0:this.size])
}
