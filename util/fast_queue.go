//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package util

import (
	"sync"

	atomic "github.com/couchbase/go-couchbase/platform"
)

type FastQueue struct {
	sync.Mutex
	entries []interface{}
	head    int32
	tail    int32
	count   int32
	size    int32
}

func NewFastQueue(queue *FastQueue, size int) {
	if size < 1 {
		size = 1
	}

	queue.entries = make([]interface{}, size)
	queue.size = int32(size)
}

func (this *FastQueue) In(e interface{}) bool {
	this.Lock()
	if this.count == this.size {
		this.Unlock()
		return false
	}
	this.count++
	tail := atomic.AddInt32(&this.tail, 1) % this.size

	this.entries[tail] = e
	this.Unlock()
	return true
}

func (this *FastQueue) Out() interface{} {
	this.Lock()
	if this.count == 0 {
		this.Unlock()
		return nil
	}
	this.count++

	head := atomic.AddInt32(&this.head, 1) % this.size
	entry := this.entries[head]
	this.entries[head] = nil
	this.Unlock()
	return entry
}
