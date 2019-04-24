//  Copyright (c) 2017 Couchbase, Inc.
//
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package util

// We implement here sync types that in sync don't do exactly what we want.
// Our implementation tends to be leaner too.

import (
	"sync"

	atomic "github.com/couchbase/go-couchbase/platform"
)

// Once is an object that will perform exactly one action.
type Once struct {
	done uint32
}

// Do calls the function f if and only if Do is being called for the
// first time for this instance of Once. In other words, given
// 	var once Once
// if once.Do(f) is called multiple times, only the first call will invoke f,
// even if f has a different value in each invocation. A new instance of
// Once is required for each function to execute.
//
// Do is intended for initialization that must be run exactly once. Since f
// is niladic, it may be necessary to use a function literal to capture the
// arguments to a function to be invoked by Do:
// 	config.once.Do(func() { config.init(filename) })
//
// Because no call to Do returns until the one call to f returns, if f causes
// Do to be called, it will deadlock.
//
// If f panics, Do considers it to have returned; future calls of Do return
// without calling f.
//
// Our Once type can be reset
func (o *Once) Do(f func()) {
	if atomic.LoadUint32(&o.done) > 0 {
		return
	}

	// Slow-path.
	if atomic.AddUint32(&o.done, 1) == 1 {
		f()
	}
}

func (o *Once) Reset() {
	atomic.StoreUint32(&o.done, 0)
}

const _POOL_SIZE = 16
const _POOL_LOAD = 16
const _POOL_MAX = 1024

type FastPool struct {
	getNext   uint32
	putNext   uint32
	load      uint32
	useCount  int32
	freeCount int32
	f         func() interface{}
	pool      [_POOL_SIZE]poolList
	free      [_POOL_SIZE]poolList
}

type poolList struct {
	head *poolEntry
	tail *poolEntry
	sync.Mutex
}

type poolEntry struct {
	entry interface{}
	next  *poolEntry
}

func NewFastPool(p *FastPool, f func() interface{}) {
	*p = FastPool{}
	p.f = f
}

func (p *FastPool) Get() interface{} {
//	if atomic.LoadInt32(&p.useCount) == 0 ||
//		atomic.LoadUint32(&p.getNext)-atomic.LoadUint32(&p.load) > _POOL_LOAD {
	if atomic.LoadUint32(&p.getNext)-atomic.LoadUint32(&p.load) > _POOL_LOAD {
		return p.f()
	}
	l := atomic.AddUint32(&p.getNext, 1) % _POOL_SIZE
	e := p.pool[l].Get()
	atomic.AddUint32(&p.load, 1)
	if e == nil {
		return p.f()
	}
	atomic.AddInt32(&p.useCount, -1)
	rv := e.entry
	e.entry = nil
	if atomic.LoadInt32(&p.freeCount) < _POOL_MAX {
		atomic.AddInt32(&p.freeCount, 1)
		p.free[l].Put(e)
	}
	return rv
}

func (p *FastPool) Put(s interface{}) {
	if atomic.LoadInt32(&p.useCount) >= _POOL_MAX {
		return
	}
	l := atomic.AddUint32(&p.putNext, 1) % _POOL_SIZE
	e := p.free[l].Get()
	if e == nil {
		e = &poolEntry{}
	} else {
		atomic.AddInt32(&p.freeCount, -1)
	}
	e.entry = s
	p.pool[l].Put(e)
	atomic.AddInt32(&p.useCount, 1)
}

func (l *poolList) Get() *poolEntry {
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

func (l *poolList) Put(e *poolEntry) {
	l.Lock()
	if l.head == nil {
		l.head = e
	} else {
		l.tail.next = e
	}
	l.tail = e
	l.Unlock()
}
