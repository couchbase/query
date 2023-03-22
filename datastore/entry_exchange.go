//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package datastore

import (
	"sync"
	"sync/atomic"
)

type scanQueue struct {
	next *scanState
	prev *scanState
}

type entryQueue struct {
	items        []*IndexEntry
	itemsHead    int
	itemsTail    int
	itemsCount   int
	closed       bool
	readWaiters  scanQueue
	writeWaiters scanQueue
	localEntries [1]*IndexEntry
	vLock        sync.Mutex
	count        int32
}

type scanState struct {
	stop       bool
	timeout    bool
	queue      scanQueue
	oLock      sync.RWMutex
	mustSignal bool
	wg         sync.WaitGroup
}

type EntryExchange struct {
	entryQueue
	reader scanState
	writer scanState
}

var entrySlicePool = sync.Pool{New: func() interface{} {
	return make([]*IndexEntry, GetScanCap())
},
}

// constructor
func newEntryExchange(exchange *EntryExchange, capacity int64) {
	if capacity <= 1 {
		capacity = 1
	}
	if capacity == 1 {
		exchange.items = exchange.localEntries[0:1:1]
	} else if capacity == GetScanCap() {
		items := entrySlicePool.Get().([]*IndexEntry)
		newCap := cap(items)
		exchange.items = items[0:newCap]
	}

	// either non standard scan cap, or server wide scan cap changes
	// and we are still caching old slices
	if exchange.items == nil || int64(cap(exchange.items)) != capacity {
		exchange.items = make([]*IndexEntry, capacity)
	}
}

// back to factory defaults
// it's the responsibility of the caller to know that no more readers or
// writers are around
func (this *EntryExchange) reset() {
	this.reader.stop = false
	this.writer.stop = false
	this.closed = false
	for this.itemsCount > 0 {
		this.items[this.itemsTail] = nil
		this.itemsCount--
		this.itemsTail++
		if this.itemsTail >= cap(this.items) {
			this.itemsTail = 0
		}
	}
	this.itemsHead = 0
	this.itemsTail = 0
}

// ditch the slices
func (this *EntryExchange) dispose() {

	// ditch entrys before pooling
	for this.itemsCount > 0 {
		this.items[this.itemsTail] = nil
		this.itemsCount--
		this.itemsTail++
		if this.itemsTail >= cap(this.items) {
			this.itemsTail = 0
		}
	}

	c := cap(this.items)
	if int64(c) == GetScanCap() {

		// scan cap might have changed in the interim
		// if ths is the case, we don't want to pool this slice
		entrySlicePool.Put(this.items[0:0])
	}
	this.items = nil
}

// capacity
func (this *EntryExchange) Capacity() int {
	return cap(this.entryQueue.items)
}

// length
func (this *EntryExchange) Length() int {
	return this.entryQueue.itemsCount
}

// send
func (this *EntryExchange) SendEntry(item *IndexEntry) bool {
	if this.writer.stop {
		return false
	}
	this.vLock.Lock()
	this.writer.oLock.Lock()
	for {

		// stop takes precedence
		if this.writer.stop || this.writer.timeout {
			this.writer.timeout = false
			this.writer.oLock.Unlock()
			this.vLock.Unlock()
			return false
		}

		// depart from channels: closed means stopped rather than panic
		// operators don't send on a closed channel anyway, so mooth
		if this.closed {
			this.readWaiters.signal()
			this.writeWaiters.signal()
			this.writer.oLock.Unlock()
			this.vLock.Unlock()
			return false
		}
		if this.itemsCount < cap(this.items) {
			break
		}
		this.writer.enqueue(this, &this.writeWaiters)
	}
	this.writer.oLock.Unlock()
	this.items[this.itemsHead] = item
	this.itemsHead++
	if this.itemsHead >= cap(this.items) {
		this.itemsHead = 0
	}
	this.itemsCount++
	this.readWaiters.signal()
	if this.itemsCount < cap(this.items) {
		this.writeWaiters.signal()
	}
	this.vLock.Unlock()

	return true
}

// receive
func (this *EntryExchange) GetEntry() (*IndexEntry, bool) {

	if this.reader.stop {
		return nil, false
	}
	this.vLock.Lock()
	this.reader.oLock.Lock()
	for {

		// stop takes precedence
		if this.reader.stop || this.reader.timeout {
			this.reader.timeout = false
			this.reader.oLock.Unlock()
			this.vLock.Unlock()
			return nil, false
		}

		if this.itemsCount > 0 {
			break
		}

		// no more
		if this.closed {
			this.reader.oLock.Unlock()
			this.readWaiters.signal()
			this.writeWaiters.signal()
			this.vLock.Unlock()
			return nil, true
		}
		this.reader.enqueue(this, &this.readWaiters)
	}
	this.reader.oLock.Unlock()
	val := this.items[this.itemsTail]
	this.items[this.itemsTail] = nil
	this.itemsTail++
	if this.itemsTail >= cap(this.items) {
		this.itemsTail = 0
	}
	this.itemsCount--
	this.writeWaiters.signal()
	if this.itemsCount > 0 {
		this.readWaiters.signal()
	}
	this.vLock.Unlock()
	return val, true
}

// append operator to correct waiter queue, wait, remove from queue
// both locks acquired in and out
func (this *scanState) enqueue(op *EntryExchange, q *scanQueue) {

	// prepare ouservelves to be woken up
	// needs to be done before adding ourselves to the queue
	this.wg.Add(1)
	this.mustSignal = true

	// append to queue
	this.queue.prev = q.prev
	this.queue.next = nil

	// fine to manipulate others queue element without acquiring
	// their oLock as they are stuck in the queue and we own the
	// queue lock
	if q.prev != nil {
		q.prev.queue.next = this
	}
	q.prev = this

	if q.next == nil {
		q.next = this
	}

	// unlock entryQueue and wait
	this.oLock.Unlock()
	op.vLock.Unlock()
	this.wg.Wait()

	// lock entryQueue and remove
	op.vLock.Lock()
	this.oLock.Lock()
	if this.queue.prev != nil {
		this.queue.prev.queue.next = this.queue.next
	}
	if this.queue.next != nil {
		this.queue.next.queue.prev = this.queue.prev
	}
	if q.next == this {
		q.next = this.queue.next
	}
	if q.prev == this {
		q.prev = this.queue.prev
	}
	this.queue.next = nil
	this.queue.prev = nil
}

func (this *scanQueue) signal() {
	if this.next != nil {
		this.next.oLock.Lock()
		if this.next.mustSignal {
			this.next.mustSignal = false
			this.next.wg.Done()
		}
		this.next.oLock.Unlock()
	}
}

// last orders!
// we expect 2 closes, 1 from the reader, 1 from the writer
// first close (from whoever) means no more data
// second close means no other active party and the connection can be disposed of
func (this *EntryExchange) Close() {
	c := atomic.AddInt32(&this.count, 1)
	switch c {
	case 1:
		this.entryQueue.close()
	case 2:
		this.dispose()
	}
}

func (this *EntryExchange) IsClosed() bool {
	return this.closed
}

func (this *entryQueue) close() {
	this.vLock.Lock()
	this.closed = true

	// wake any readers and writers
	this.readWaiters.signal()
	this.writeWaiters.signal()
	this.vLock.Unlock()
}

// signal stop
func (this *EntryExchange) sendStop() {
	this.reader.sendStop()
	this.writer.sendStop()
}

func (this *scanState) sendStop() {
	this.oLock.Lock()
	this.stop = true
	if this.mustSignal {
		this.mustSignal = false
		this.wg.Done()
	}
	this.oLock.Unlock()
}

// signal sender timeout
func (this *EntryExchange) sendTimeout() {
	this.writer.sendTimeout()
}

func (this *scanState) sendTimeout() {
	this.oLock.Lock()
	this.timeout = true
	if this.mustSignal {
		this.mustSignal = false
		this.wg.Done()
	}
	this.oLock.Unlock()
}

// did we get a stop?
func (this *EntryExchange) IsStopped() bool {
	this.reader.oLock.RLock()
	rv := this.reader.stop
	this.reader.oLock.RUnlock()
	return rv
}
