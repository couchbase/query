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

type SetPool struct {
	pool      util.FastPool
	objectCap int
	collect   bool
}

// numeric is a flag to restrict the Set to only contain numeric values(float64 and int64).
func NewSetPool(objectCap int, collect, numeric bool) *SetPool {
	rv := &SetPool{
		objectCap: objectCap,
		collect:   collect,
	}
	util.NewFastPool(&rv.pool, func() interface{} {
		return NewSet(objectCap, collect, numeric)
	})
	return rv
}

func (this *SetPool) Get() *Set {
	return this.pool.Get().(*Set)
}

func (this *SetPool) Put(s *Set) {
	if s.Len() > 16*this.objectCap {
		return
	}

	s.Clear()
	this.pool.Put(s)
}
