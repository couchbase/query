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
	"runtime"
	"sync"

	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

const _SMALL_CHILD_POOL = 4
const _LARGE_CHILD_POOL = 64
const _MAX_THRESHOLD = 16

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
	size         uint64
	maxSize      uint64
	memYields    int
	heartbeat    int
	beatYields   int
	closed       bool
	readWaiters  opQueue
	writeWaiters opQueue
	localValues  [1]value.AnnotatedValue
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

var valueSlicePool util.FastPool

var smallChildPool util.FastPool

var largeChildPool util.FastPool

var threshold int

func init() {
	util.NewFastPool(&valueSlicePool, func() interface{} {
		return make([]value.AnnotatedValue, GetPipelineCap())
	})
	util.NewFastPool(&smallChildPool, func() interface{} {
		return make([]int, _SMALL_CHILD_POOL)
	})
	util.NewFastPool(&largeChildPool, func() interface{} {
		return make([]int, _LARGE_CHILD_POOL)
	})
	threshold = runtime.NumCPU()
	if threshold > _MAX_THRESHOLD {
		threshold = _MAX_THRESHOLD
	}
}

// constructor
func newValueExchange(exchange *valueExchange, capacity int64) {
	if capacity <= 1 {
		capacity = 1
	}
	if capacity == 1 {
		exchange.items = exchange.localValues[0:1:1]
	} else if capacity == GetPipelineCap() {
		items := valueSlicePool.Get().([]value.AnnotatedValue)
		newCap := cap(items)
		exchange.items = items[0:newCap]
	}

	// either non standard pipeline cap, or server wide pipeline cap changes
	// and we are still caching old slices
	if exchange.items == nil || int64(cap(exchange.items)) != capacity {
		exchange.items = make([]value.AnnotatedValue, capacity)
	}
	exchange.size = 0
	exchange.maxSize = 0
	exchange.heartbeat = 0
	exchange.memYields = 0
	exchange.beatYields = 0
}

func (this *valueExchange) cap() int {
	return cap(this.items)
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
		this.items[this.itemsTail].Recycle()
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
	this.size = 0
	// not maxSize, no yields
	this.heartbeat = 0
}

// ditch the slices
func (this *valueExchange) dispose() {

	// MB-28710 ditch values before pooling
	for this.itemsCount > 0 {
		this.items[this.itemsTail].Recycle()
		this.items[this.itemsTail] = nil
		this.itemsCount--
		this.itemsTail++
		if this.itemsTail >= cap(this.items) {
			this.itemsTail = 0
		}
	}

	c := cap(this.items)
	if c > 1 && int64(c) == GetPipelineCap() {

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

// present a different operator with our value exchange and
// set a default for ours
// not to be use on an already allocated / used dest!
// only to be use before dest is allocated (let alone used)!
func (this *valueExchange) move(dest *valueExchange) {
	if cap(this.items) == 1 {
		dest.items = dest.localValues[0:1:1]
	} else {
		*dest = *this
		this.items = this.localValues[0:1:1]
	}
}

// send
func (this *valueExchange) sendItem(op *valueExchange, item value.AnnotatedValue, quota uint64) bool {
	if this.stop {
		return false
	}

	op.vLock.Lock()
	this.oLock.Lock()
	if quota > 0 {
		op.size += item.Size()
		if op.size > op.maxSize {
			op.maxSize = op.size
		}
	}
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

		// In order to avoid a stall, we won't throttle when no document
		// is queued, even it it exceeds any throttling threshold
		if op.itemsCount == 0 {
			break
		}
		if op.itemsCount >= cap(op.items) {
			this.enqueue(op, &op.writeWaiters)
			continue
		}
		if op.readWaiters.next != nil {
			op.heartbeat++

			// give precendence to the consumer if it didn't get a chance
			// to run in a timely manner
			// ideally we would want to yield to the first waiter, but
			// golang does not allow that choice
			if op.heartbeat > threshold {
				this.beatYields++
				this.enqueue(op, &op.writeWaiters)
				continue
			}
		}
		if quota != 0 && op.size > quota {
			this.memYields++
			this.enqueue(op, &op.writeWaiters)
			continue
		}
		break
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
	if op.size > 0 {
		op.size -= val.Size()
	}
	op.heartbeat = 0
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
	if op.size > 0 {
		op.size -= val.Size()
	}
	op.heartbeat = 0
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

// are we waiting?
func (this *valueExchange) isWaiting() bool {
	this.oLock.Lock()
	rv := this.mustSignal
	this.oLock.Unlock()
	return rv
}
