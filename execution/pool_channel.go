//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"github.com/couchbase/query/util"
)

type ChannelPool struct {
	pool util.FastPool
	size int
}

func NewChannelPool(size int) *ChannelPool {
	rv := &ChannelPool{
		size: size,
	}
	util.NewFastPool(&rv.pool, func() interface{} {
		return make([]*Channel, 0, size)
	})

	return rv
}

func (this *ChannelPool) Get() []*Channel {
	return this.pool.Get().([]*Channel)
}

func (this *ChannelPool) Put(s []*Channel) {
	if cap(s) != this.size {
		return
	}

	this.pool.Put(s[0:0])
}
