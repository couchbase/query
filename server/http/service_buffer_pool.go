//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package http

import (
	"bytes"
	"sync"
)

// BufferPool provides an API for managing bytes.Buffer objects:
type BufferPool interface {
	GetBuffer() *bytes.Buffer
	PutBuffer(*bytes.Buffer)
	BufferCapacity() int
}

// syncPoolBufPool is an implementation of BufferPool
// that uses a sync.Pool to maintain buffers:
type syncPoolBufPool struct {
	pool       *sync.Pool
	buf_size   int
	makeBuffer func() interface{}
}

func NewSyncPool(buf_size int) BufferPool {
	var newPool syncPoolBufPool

	newPool.makeBuffer = func() interface{} {
		var b bytes.Buffer

		// the buffer pool will eventually home just
		// KeepAlive size buffers, so we just as well
		// start from that
		b.Grow(buf_size)
		return &b
	}
	newPool.pool = &sync.Pool{}
	newPool.pool.New = newPool.makeBuffer
	newPool.buf_size = buf_size

	return &newPool
}

func (bp *syncPoolBufPool) GetBuffer() *bytes.Buffer {
	return bp.pool.Get().(*bytes.Buffer)
}

func (bp *syncPoolBufPool) PutBuffer(b *bytes.Buffer) {
	b.Reset()
	bp.pool.Put(b)
}

func (bp *syncPoolBufPool) BufferCapacity() int {
	return bp.buf_size
}
