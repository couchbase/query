//  Copyright 2020-Present Couchbase, Inc.
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

const _MAX_POOLED = 128

type GoroutinePool struct {
	getNext  uint32
	putNext  uint32
	queues   []opQueue
	buckets  uint32
	useCount int32
	maxCount int32
	f        func(interface{})
}

type opQueue struct {
	next *operatorState
	prev *operatorState
	sync.Mutex
}

type operatorState struct {
	args       interface{}
	oLock      sync.Mutex
	mustSignal bool
	wg         sync.WaitGroup
	next       *operatorState
	prev       *operatorState
}

// this is used for sending operation messages
// currently empty because all we do is stop the goroutine
type opCommand struct {
}

func NewGoroutinePool(p *GoroutinePool, f func(interface{})) {
	*p = GoroutinePool{}
	p.f = f
	p.buckets = uint32(NumCPU())
	if p.buckets > _MAX_BUCKETS {
		p.buckets = _MAX_BUCKETS
	} else if p.buckets < _MIN_BUCKETS {
		p.buckets = _MIN_BUCKETS
	}
	p.maxCount = int32(p.buckets) * 4
	p.queues = make([]opQueue, p.buckets)
}

func (p *GoroutinePool) Exec(args interface{}) {
	if atomic.LoadInt32(&p.useCount) == 0 {
		go p.runner(&operatorState{args: args})
		return
	}

	l := atomic.AddUint32(&p.getNext, 1) % p.buckets
	p.queues[l].Lock()
	if p.queues[l].next == nil {
		p.queues[l].Unlock()
		go p.runner(&operatorState{args: args})
		return
	}

	// extract from queue
	o := p.queues[l].next
	p.queues[l].next = o.next
	if p.queues[l].prev == o {
		p.queues[l].prev = nil
	}
	atomic.AddInt32(&p.useCount, -1)
	p.queues[l].Unlock()

	// wake up
	o.oLock.Lock()
	o.args = args
	if o.mustSignal {
		o.mustSignal = false
		o.wg.Done()
	}
	o.oLock.Unlock()
}

// this must be used when it is known that no goroutine is running
func (p *GoroutinePool) Close() {
	l := uint32(0)
	for atomic.LoadInt32(&p.useCount) > 0 {
		p.queues[l].Lock()
		if p.queues[l].next == nil {
			p.queues[l].Unlock()
			l = (l + 1) % (p.buckets)
			continue
		}

		// extract from queue
		o := p.queues[l].next
		p.queues[l].next = o.next
		if p.queues[l].prev == o {
			p.queues[l].prev = nil
		}

		atomic.AddInt32(&p.useCount, -1)
		p.queues[l].Unlock()

		// wake up and instruct to terminate
		o.oLock.Lock()
		o.args = &opCommand{}
		if o.mustSignal {
			o.mustSignal = false
			o.wg.Done()
		}
		o.oLock.Unlock()
	}
}

func (p *GoroutinePool) runner(state *operatorState) {
	for {
		args := state.args
		state.args = nil

		// have we been terminated?
		_, ok := args.(*opCommand)
		if ok {
			return
		}

		// execute the work
		p.f(args)

		// if enough goroutines pooled, our services are no longer needed`
		if !state.enqueue(p) {
			return
		}
	}
}

func (this *operatorState) enqueue(p *GoroutinePool) bool {
	if atomic.LoadInt32(&p.useCount) >= p.maxCount {
		return false
	}
	this.oLock.Lock()

	// prepare ouservelves to be woken up
	// needs to be done before adding ourselves to the queue
	this.wg.Add(1)
	this.mustSignal = true
	this.oLock.Unlock()

	l := atomic.AddUint32(&p.putNext, 1) % p.buckets
	p.queues[l].Lock()

	// append to queue
	this.prev = p.queues[l].prev
	this.next = nil

	if p.queues[l].prev != nil {
		p.queues[l].prev.next = this
	}
	p.queues[l].prev = this

	if p.queues[l].next == nil {
		p.queues[l].next = this
	}
	atomic.AddInt32(&p.useCount, 1)

	// unlock the queue and wait
	p.queues[l].Unlock()
	this.wg.Wait()

	return true
}
