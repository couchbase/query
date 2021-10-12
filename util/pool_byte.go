//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package util

import (
	"sync"

	atomic "github.com/couchbase/go-couchbase/platform"
)

type BytePool struct {
	pool byteSliceFastPool
	size int
}

func NewBytePool(size int) *BytePool {
	rv := &BytePool{
		size: size,
	}
	newByteSliceFastPool(&rv.pool, func() []byte {
		return make([]byte, 0, rv.size)
	})

	return rv
}

func (this *BytePool) Get() []byte {
	return this.pool.get()
}

func (this *BytePool) GetCapped(capacity int) []byte {
	if capacity > this.size {
		return make([]byte, 0, capacity)
	} else {
		return this.Get()
	}
}

func (this *BytePool) GetSized(length int) []byte {
	if length > this.size {
		return make([]byte, length)
	}

	rv := this.Get()
	rv = rv[0:length]
	for i := 0; i < length; i++ {
		rv[i] = byte(0)
	}

	return rv
}

func (this *BytePool) Put(b []byte) {
	if cap(b) != this.size {
		return
	}

	this.pool.put(b[0:0])
}

// this is a type-specific implementation of FastPool to avoid implicit memory allocation for conversion from slice to interface{}

type byteSliceFastPool struct {
	getNext   uint32
	putNext   uint32
	useCount  int32
	freeCount int32
	buckets   uint32
	f         func() []byte
	pool      []byteSlicePoolList
	free      []byteSlicePoolList
}

type byteSlicePoolList struct {
	head *byteSlicePoolEntry
	tail *byteSlicePoolEntry
	sync.Mutex
}

type byteSlicePoolEntry struct {
	entry []byte
	next  *byteSlicePoolEntry
}

func newByteSliceFastPool(p *byteSliceFastPool, f func() []byte) {
	*p = byteSliceFastPool{}
	p.buckets = uint32(NumCPU())
	if p.buckets > _MAX_BUCKETS {
		p.buckets = _MAX_BUCKETS
	} else if p.buckets < _MIN_BUCKETS {
		p.buckets = _MIN_BUCKETS
	}
	p.pool = make([]byteSlicePoolList, p.buckets)
	p.free = make([]byteSlicePoolList, p.buckets)
	p.f = f
}

func (p *byteSliceFastPool) get() []byte {
	if atomic.LoadInt32(&p.useCount) == 0 {
		return p.f()
	}
	l := atomic.AddUint32(&p.getNext, 1) % p.buckets
	e := p.pool[l].get()
	if e == nil {
		return p.f()
	}
	atomic.AddInt32(&p.useCount, -1)
	rv := e.entry
	e.entry = nil
	if atomic.LoadInt32(&p.freeCount) < _POOL_SIZE {
		atomic.AddInt32(&p.freeCount, 1)
		p.free[l].put(e)
	}
	return rv
}

func (p *byteSliceFastPool) put(s []byte) {
	if atomic.LoadInt32(&p.useCount) >= _POOL_SIZE {
		return
	}
	l := atomic.AddUint32(&p.putNext, 1) % p.buckets
	e := p.free[l].get()
	if e == nil {
		e = &byteSlicePoolEntry{}
	} else {
		atomic.AddInt32(&p.freeCount, -1)
	}
	e.entry = s
	p.pool[l].put(e)
	atomic.AddInt32(&p.useCount, 1)
}

func (l *byteSlicePoolList) get() *byteSlicePoolEntry {
	if l.head == nil {
		return nil
	}

	l.Lock()
	if l.head == nil {
		l.Unlock()
		return nil
	}
	rv := l.head
	l.head = rv.next
	l.Unlock()
	rv.next = nil
	return rv
}

func (l *byteSlicePoolList) put(e *byteSlicePoolEntry) {
	l.Lock()
	if l.head == nil {
		l.head = e
	} else {
		l.tail.next = e
	}
	l.tail = e
	l.Unlock()
}
