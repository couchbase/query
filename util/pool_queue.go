//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package util

type QueuePool struct {
	pool FastPool
	size int
}

func NewQueuePool(size int) *QueuePool {
	rv := &QueuePool{
		size: size,
	}
	NewFastPool(&rv.pool, func() interface{} {
		return NewQueue(size)
	})

	return rv
}

func (this *QueuePool) Get() *Queue {
	return this.pool.Get().(*Queue)
}

func (this *QueuePool) Put(s *Queue) {
	if s.Capacity() != this.size {
		return
	}

	s.Clear()
	this.pool.Put(s)
}
