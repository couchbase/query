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

type BagPool struct {
	pool      util.FastPool
	objectCap int
}

func NewBagPool(objectCap int) *BagPool {
	rv := &BagPool{
		objectCap: objectCap,
	}
	util.NewFastPool(&rv.pool, func() interface{} {
		return NewBag(objectCap)
	})
	return rv
}

func (this *BagPool) Get() *Bag {
	return this.pool.Get().(*Bag)
}

func (this *BagPool) Put(s *Bag) {
	if s.DistinctLen() > 16*this.objectCap {
		return
	}

	s.Clear()
	this.pool.Put(s)
}
