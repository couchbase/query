//  Copyright (c) 2021 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package util

import (
	"runtime"
	"sync"

	atomic "github.com/couchbase/go-couchbase/platform"
)

type BytePool struct {
	pool ByteSliceFastPool
	size int
}

func NewBytePool(size int) *BytePool {
	rv := &BytePool{
		size: size,
	}
	NewByteSliceFastPool(&rv.pool, func() []byte {
		return make([]byte, 0, rv.size)
	})

	return rv
}

func (this *BytePool) Get() []byte {
	return this.pool.Get()
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

	this.pool.Put(b[0:0])
}

// this is a type-specific implementation of FastPool to avoid implicit memory allocation for conversion from slice to interface{}

type ByteSliceFastPool struct {
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

func NewByteSliceFastPool(p *ByteSliceFastPool, f func() []byte) {
	*p = ByteSliceFastPool{}
	p.buckets = uint32(runtime.NumCPU())
	if p.buckets > _MAX_BUCKETS {
		p.buckets = _MAX_BUCKETS
	} else if p.buckets < _MIN_BUCKETS {
		p.buckets = _MIN_BUCKETS
	}
	p.pool = make([]byteSlicePoolList, p.buckets)
	p.free = make([]byteSlicePoolList, p.buckets)
	p.f = f
}

func (p *ByteSliceFastPool) Get() []byte {
	if atomic.LoadInt32(&p.useCount) == 0 {
		return p.f()
	}
	l := atomic.AddUint32(&p.getNext, 1) % p.buckets
	e := p.pool[l].Get()
	if e == nil {
		return p.f()
	}
	atomic.AddInt32(&p.useCount, -1)
	rv := e.entry
	e.entry = nil
	if atomic.LoadInt32(&p.freeCount) < _POOL_SIZE {
		atomic.AddInt32(&p.freeCount, 1)
		p.free[l].Put(e)
	}
	return rv
}

func (p *ByteSliceFastPool) Put(s []byte) {
	if atomic.LoadInt32(&p.useCount) >= _POOL_SIZE {
		return
	}
	l := atomic.AddUint32(&p.putNext, 1) % p.buckets
	e := p.free[l].Get()
	if e == nil {
		e = &byteSlicePoolEntry{}
	} else {
		atomic.AddInt32(&p.freeCount, -1)
	}
	e.entry = s
	p.pool[l].Put(e)
	atomic.AddInt32(&p.useCount, 1)
}

func (l *byteSlicePoolList) Get() *byteSlicePoolEntry {
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

func (l *byteSlicePoolList) Put(e *byteSlicePoolEntry) {
	l.Lock()
	if l.head == nil {
		l.head = e
	} else {
		l.tail.next = e
	}
	l.tail = e
	l.Unlock()
}
