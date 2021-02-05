//  Copyright (c) 2014 Couchbase, Inc.
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

type StringPool struct {
	pool stringSliceFastPool
	size int
}

func NewStringPool(size int) *StringPool {
	rv := &StringPool{
		size: size,
	}
	NewstringSliceFastPool(&rv.pool, func() []string {
		return make([]string, 0, rv.size)
	})

	return rv
}

func (this *StringPool) Get() []string {
	return this.pool.get()
}

func (this *StringPool) GetCapped(capacity int) []string {
	if capacity > this.size {
		return make([]string, 0, capacity)
	} else {
		return this.Get()
	}
}

func (this *StringPool) GetSized(length int) []string {
	if length > this.size {
		return make([]string, length)
	}

	rv := this.Get()
	rv = rv[0:length]
	for i := 0; i < length; i++ {
		rv[i] = ""
	}

	return rv
}

func (this *StringPool) Put(s []string) {
	if cap(s) != this.size {
		return
	}

	this.pool.put(s[0:0])
}

// this is a type-specific implementation of FastPool to avoid implicit memory allocation for conversion from slice to interface{}

type stringSliceFastPool struct {
	getNext   uint32
	putNext   uint32
	useCount  int32
	freeCount int32
	buckets   uint32
	f         func() []string
	pool      []stringSlicePoolList
	free      []stringSlicePoolList
}

type stringSlicePoolList struct {
	head *stringSlicePoolEntry
	tail *stringSlicePoolEntry
	sync.Mutex
}

type stringSlicePoolEntry struct {
	entry []string
	next  *stringSlicePoolEntry
}

func NewstringSliceFastPool(p *stringSliceFastPool, f func() []string) {
	*p = stringSliceFastPool{}
	p.buckets = uint32(runtime.NumCPU())
	if p.buckets > _MAX_BUCKETS {
		p.buckets = _MAX_BUCKETS
	} else if p.buckets < _MIN_BUCKETS {
		p.buckets = _MIN_BUCKETS
	}
	p.pool = make([]stringSlicePoolList, p.buckets)
	p.free = make([]stringSlicePoolList, p.buckets)
	p.f = f
}

func (p *stringSliceFastPool) get() []string {
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

func (p *stringSliceFastPool) put(s []string) {
	if atomic.LoadInt32(&p.useCount) >= _POOL_SIZE {
		return
	}
	l := atomic.AddUint32(&p.putNext, 1) % p.buckets
	e := p.free[l].get()
	if e == nil {
		e = &stringSlicePoolEntry{}
	} else {
		atomic.AddInt32(&p.freeCount, -1)
	}
	e.entry = s
	p.pool[l].put(e)
	atomic.AddInt32(&p.useCount, 1)
}

func (l *stringSlicePoolList) get() *stringSlicePoolEntry {
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

func (l *stringSlicePoolList) put(e *stringSlicePoolEntry) {
	l.Lock()
	if l.head == nil {
		l.head = e
	} else {
		l.tail.next = e
	}
	l.tail = e
	l.Unlock()
}
