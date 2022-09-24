//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package util

// We implement here sync types that in sync don't do exactly what we want.
// Our implementation tends to be leaner too.

import (
	"fmt"
	"sync"
	"time"
	"unsafe"

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

const _MIN_BUCKETS = 8
const _MAX_BUCKETS = 64
const _POOL_SIZE = 1024

type FastPool struct {
	getNext   uint32
	putNext   uint32
	useCount  int32
	freeCount int32
	buckets   uint32
	f         func() interface{}
	pool      []poolList
	free      []poolList
	interval  time.Duration
	low       int32
	high      int32
	bailOut   chan bool
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
	p.buckets = uint32(NumCPU())
	if p.buckets > _MAX_BUCKETS {
		p.buckets = _MAX_BUCKETS
	} else if p.buckets < _MIN_BUCKETS {
		p.buckets = _MIN_BUCKETS
	}
	p.pool = make([]poolList, p.buckets)
	p.free = make([]poolList, p.buckets)
	p.f = f
}

func (p *FastPool) Get() interface{} {
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

func (p *FastPool) Put(s interface{}) {
	if atomic.LoadInt32(&p.useCount) >= _POOL_SIZE {
		return
	}
	l := atomic.AddUint32(&p.putNext, 1) % p.buckets
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

// a drainer can only be started once
// close the pool and start a new one
func (p *FastPool) Drain(low, high int, interval time.Duration) error {
	if p.bailOut != nil {
		return fmt.Errorf("Draining already set up")
	}
	if interval < time.Second {
		return fmt.Errorf("Invalid interval")
	}
	if low < 0 || high > 100 || low >= high {
		return fmt.Errorf("Invalid watermarks")
	}
	p.low = int32(low * _POOL_SIZE / 100)
	p.high = int32(high * _POOL_SIZE / 100)
	p.interval = interval
	p.bailOut = make(chan bool)
	go p.drainer()
	return nil
}

// a closed pool is not supposed to be reopened
// initialize a new pool
func (p *FastPool) Close() {
	if p.bailOut != nil {

		// defensively, we won't wait if the channel is full
		select {
		case p.bailOut <- false:
		default:
		}
	}
}

// asynchronous connection closer
func (p *FastPool) drainer() {
	t := time.NewTimer(p.interval)
	defer t.Stop()

	for {
		getNext := p.getNext

		// we don't exist anymore! bail out!
		select {
		case <-p.bailOut:
			return
		case <-t.C:
		}
		t.Reset(p.interval)

		// demand for entries is there
		if p.getNext != getNext || p.useCount <= p.high {
			continue
		}

		// remove excess free entries
		for p.useCount > p.low {
			select {
			case <-p.bailOut:
				return
			default:
			}

			l := atomic.AddUint32(&p.getNext, 1) % p.buckets
			e := p.pool[l].Get()
			if e != nil {
				atomic.AddInt32(&p.useCount, -1)
				e.entry = nil
				if atomic.LoadInt32(&p.freeCount) < _POOL_SIZE {
					atomic.AddInt32(&p.freeCount, 1)
					p.free[l].Put(e)
				}
			}
		}
	}
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

/*
 * We use non-atomic operations on the counters to reduce contention.  This does mean that we may skip caching values and/or have
 * a sparsely populated array, but overall with enough pressure we cache sufficient elements to be of value still whilst not
 * penalising the throughput.  The only operation that must be atomic is the acquisition of an element from the pool.
 */
type LocklessPool struct {
	getNext uint32
	putNext uint32
	f       func() unsafe.Pointer
	pool    [_POOL_SIZE]unsafe.Pointer
}

func NewLocklessPool(p *LocklessPool, f func() unsafe.Pointer) {
	*p = LocklessPool{}
	p.f = f
	p.getNext = _POOL_SIZE / 2
}

func (p *LocklessPool) Get() unsafe.Pointer {
	l := p.getNext % _POOL_SIZE
	e := atomic.SwapPointer(&p.pool[l], nil) // must be atomic to prevent multiple users

	// niet
	if e == nil {
		return p.f()
	} else {
		p.getNext++
		return e
	}

	return e
}

func (p *LocklessPool) Put(s unsafe.Pointer) {
	l := p.putNext % _POOL_SIZE
	p.putNext++
	atomic.StorePointer(&p.pool[l], s)
}

type WaitCount struct {
	count int32
}

func (this *WaitCount) Value() int32 {
	return atomic.LoadInt32(&this.count)
}

func (this *WaitCount) Incr() {
	atomic.AddInt32(&this.count, 1)
}

func (this *WaitCount) Decr() {
	atomic.AddInt32(&this.count, -1)
}

func (this *WaitCount) Set(v int32) {
	atomic.StoreInt32(&this.count, v)
}

var _WAIT_TIME = 100 * time.Millisecond

func (this *WaitCount) Until(v int32, limit time.Duration) bool {
	useTimeout := limit.Nanoseconds() != 0
	start := Now()
	for !atomic.CompareAndSwapInt32(&this.count, v, this.count) {
		if useTimeout && Since(start).Nanoseconds() > limit.Nanoseconds() {
			return false
		}
		time.Sleep(_WAIT_TIME)
	}
	return true
}
