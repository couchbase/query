//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execution

import (
	"sync"

	"github.com/couchbase/query/value"
)

const _SMALL_CHILD_POOL = 4
const _LARGE_CHILD_POOL = 64

// A couple of design considerations: we have to be able to send and receive items from any
// item channel, but get notified of a stop message or a child specifically on our own.
// This means that we need to have two separate structures (one possibly shared and residing
// elsewhere and one our own) that must be operated upon at the same time.
// In order to avoid deadlocks, any actor will just signal waiters as required, but manipulate
// nothing bar changing whatever state is required.
// It is the responsibility of the newly woken go routine to manipulate the queue as required.

type opQueue struct {
	next *operatorState
	prev *operatorState
}

type valueQueue struct {
	items        []value.AnnotatedValue
	itemsHead    int
	itemsTail    int
	itemsCount   int
	closed       bool
	readWaiters  opQueue
	writeWaiters opQueue
	vLock        sync.Mutex
}

type operatorState struct {
	stop       bool
	children   []int
	queue      opQueue
	oLock      sync.Mutex
	mustSignal bool
	wg         sync.WaitGroup
}

type valueExchange struct {
	valueQueue
	operatorState
}

var valueSlicePool = sync.Pool{New: func() interface{} {
	return make([]value.AnnotatedValue, GetPipelineCap())
},
}

var smallSlicePool = sync.Pool{New: func() interface{} {
	return make([]value.AnnotatedValue, 1)
},
}

var smallChildPool = sync.Pool{New: func() interface{} {
	return make([]int, _SMALL_CHILD_POOL)
},
}

var largeChildPool = sync.Pool{New: func() interface{} {
	return make([]int, _LARGE_CHILD_POOL)
},
}

// constructor
func newValueExchange(exchange *valueExchange, capacity int64) {
	if capacity <= 1 {
		capacity = 1
	}
	if capacity == 1 {
		exchange.items = smallSlicePool.Get().([]value.AnnotatedValue)[0:capacity]
	} else if capacity == GetPipelineCap() {
		exchange.items = valueSlicePool.Get().([]value.AnnotatedValue)[0:capacity]
	}

	// either non standard pipeline cap, or server wide pipeline cap changes
	// and we are still caching old slices
	if exchange.items == nil || int64(cap(exchange.items)) != GetPipelineCap() {
		exchange.items = make([]value.AnnotatedValue, capacity)
	}
}

// for those operators that have children
func (this *valueExchange) trackChildren(children int) {

	// these are initial sizes for the children slices.
	// small is enough for things like merge, large for the scans
	// the slices will be appended to and therefore will grow as
	// as needed, so we don't have to check here for an upper children
	// limit
	if children > _SMALL_CHILD_POOL {
		this.children = largeChildPool.Get().([]int)[0:0]
	} else {
		this.children = smallChildPool.Get().([]int)[0:0]
	}
}

// back to factory defaults
// it's the responsibility of the caller to know that no more readers or
// writers are around
func (this *valueExchange) reset() {
	this.stop = false
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
	if this.children != nil {
		this.children = this.children[0:0]
	}
}

// ditch the slices
func (this *valueExchange) dispose() {

	// MB-28710 ditch values before pooling
	for this.itemsCount > 0 {
		this.items[this.itemsTail] = nil
		this.itemsCount--
		this.itemsTail++
		if this.itemsTail >= cap(this.items) {
			this.itemsTail = 0
		}
	}

	c := cap(this.items)
	if c == 1 {
		smallSlicePool.Put(this.items[0:0])
	} else if int64(c) == GetPipelineCap() {

		// pipeline cap might have changed in the interim
		// if ths is the case, we don't want to pool this slice
		valueSlicePool.Put(this.items[0:0])
	}
	this.items = nil

	// the slices might have grown with the appends, if the sizing was
	// approximate, so anything which started with _LARGE_CHILD_POOL
	// still goes back to the large pool, even if it has grown slightly
	// ditto for the small pool
	if this.children == nil {
		return
	} else if cap(this.children) >= _LARGE_CHILD_POOL {
		largeChildPool.Put(this.children[0:0])
	} else {
		smallChildPool.Put(this.children[0:0])
	}
	this.children = nil
}

// send
func (this *valueExchange) sendItem(op *valueExchange, item value.AnnotatedValue) bool {
	if this.stop {
		return false
	}
	op.vLock.Lock()
	this.oLock.Lock()
	for {

		// stop takes precedence
		if this.stop {
			this.oLock.Unlock()
			op.vLock.Unlock()
			return false
		}

		// depart from channels: closed means stopped rather than panic
		// operators don't send on a closed channel anyway, so mooth
		if op.closed {
			op.readWaiters.signal()
			op.writeWaiters.signal()
			this.oLock.Unlock()
			op.vLock.Unlock()
			return false
		}
		if op.itemsCount < cap(op.items) {
			break
		}
		this.enqueue(op, &op.writeWaiters)

	}
	this.oLock.Unlock()
	op.items[op.itemsHead] = item
	op.itemsHead++
	if op.itemsHead >= cap(op.items) {
		op.itemsHead = 0
	}
	op.itemsCount++
	op.readWaiters.signal()
	if op.itemsCount < cap(op.items) {
		op.writeWaiters.signal()
	}
	op.vLock.Unlock()

	return true
}

// channel length
func (this *valueExchange) queuedItems(op *valueExchange) int {
	return op.itemsCount
}

// receive
func (this *valueExchange) getItem(op *valueExchange) (value.AnnotatedValue, bool) {
	if this.stop {
		return nil, false
	}
	op.vLock.Lock()
	this.oLock.Lock()
	for {

		// stop takes precedence
		if this.stop {
			this.oLock.Unlock()
			op.vLock.Unlock()
			return nil, false
		}

		if op.itemsCount > 0 {
			break
		}

		// no more
		if op.closed {
			this.oLock.Unlock()
			op.readWaiters.signal()
			op.writeWaiters.signal()
			op.vLock.Unlock()
			return nil, true
		}
		this.enqueue(op, &op.readWaiters)
	}
	this.oLock.Unlock()
	val := op.items[op.itemsTail]
	op.items[op.itemsTail] = nil
	op.itemsTail++
	if op.itemsTail >= cap(op.items) {
		op.itemsTail = 0
	}
	op.itemsCount--
	op.writeWaiters.signal()
	if op.itemsCount > 0 {
		op.readWaiters.signal()
	}
	op.vLock.Unlock()
	return val, true
}

// receive or listen to children
func (this *valueExchange) getItemChildren(op *valueExchange) (value.AnnotatedValue, int, bool) {
	if this.stop {
		return nil, -1, false
	}
	op.vLock.Lock()
	this.oLock.Lock()
	for {

		// stop takes precedence
		if this.stop {
			this.oLock.Unlock()
			op.vLock.Unlock()
			return nil, -1, false
		}

		// then children
		if len(this.children) > 0 {
			child := this.children[0]
			this.children = this.children[1:]
			this.oLock.Unlock()
			op.vLock.Unlock()
			return nil, child, true
		}

		if op.itemsCount > 0 {
			break
		}

		// no more
		if op.closed {
			this.oLock.Unlock()
			op.readWaiters.signal()
			op.writeWaiters.signal()
			op.vLock.Unlock()
			return nil, -1, true
		}
		this.enqueue(op, &op.readWaiters)
	}
	this.oLock.Unlock()
	val := op.items[op.itemsTail]
	op.items[op.itemsTail] = nil
	op.itemsTail++
	if op.itemsTail >= cap(op.items) {
		op.itemsTail = 0
	}
	op.itemsCount--
	op.writeWaiters.signal()
	if op.itemsCount > 0 {
		op.readWaiters.signal()
	}
	op.vLock.Unlock()
	return val, -1, true
}

// append operator to correct waiter queue, wait, remove from queue
// both locks acquired in and out
func (this *operatorState) enqueue(op *valueExchange, q *opQueue) {

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

	// unlock valueQueue and wait
	this.oLock.Unlock()
	op.vLock.Unlock()
	this.wg.Wait()

	// lock valueQueue and remove
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

func (this *opQueue) signal() {
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
func (this *valueQueue) close() {
	this.vLock.Lock()
	this.closed = true

	// wake any readers and writers
	this.readWaiters.signal()
	this.writeWaiters.signal()
	this.vLock.Unlock()
}

// wait for children without stopping or receiving
// if there are no more children, this will hang
func (this *valueExchange) retrieveChildNoStop() int {
	this.oLock.Lock()
	for {
		if len(this.children) > 0 {
			child := this.children[0]
			this.children = this.children[1:]
			this.oLock.Unlock()
			return child
		}
		this.wg.Add(1)
		this.mustSignal = true
		this.oLock.Unlock()
		this.wg.Wait()
		this.oLock.Lock()
	}

	// we never get here
	return -1
}

// wait for children without stopping or receiving
func (this *valueExchange) retrieveChild() (int, bool) {
	if this.stop {
		return -1, false
	}
	this.oLock.Lock()
	for {
		if this.stop {
			this.oLock.Unlock()
			return -1, false
		}
		if len(this.children) > 0 {
			child := this.children[0]
			this.children = this.children[1:]
			this.oLock.Unlock()
			return child, true
		}
		this.wg.Add(1)
		this.mustSignal = true
		this.oLock.Unlock()
		this.wg.Wait()
		this.oLock.Lock()
	}

	// we never get here
	return -1, true
}

// child signal
func (this *valueExchange) sendChild(child int) {
	this.oLock.Lock()

	// we should have enough space here for the append
	// but if we haven't this sorts itself out
	this.children = append(this.children, child)
	if this.mustSignal {
		this.mustSignal = false
		this.wg.Done()
	}
	this.oLock.Unlock()
}

// signal stop
func (this *valueExchange) sendStop() {
	this.oLock.Lock()
	this.stop = true
	if this.mustSignal {
		this.mustSignal = false
		this.wg.Done()
	}
	this.oLock.Unlock()
}

// did we get a stop?
func (this *valueExchange) isStopped() bool {
	this.oLock.Lock()
	rv := this.stop
	this.oLock.Unlock()
	return rv
}
