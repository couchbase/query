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

	"github.com/couchbase/query/util"
)

// BufferPool provides an API for managing bytes.Buffer objects:
type BufferPool interface {
	GetBuffer() *bytes.Buffer
	PutBuffer(*bytes.Buffer)
	SetBufferCapacity(s int)
	BufferCapacity() int
}

// syncPoolBufPool is an implementation of BufferPool
// that uses a FastPool to maintain buffers:
type syncPoolBufPool struct {
	pool     util.FastPool
	buf_size int
	max_size int
}

func NewSyncPool(buf_size int) BufferPool {
	newPool := &syncPoolBufPool{}
	util.NewFastPool(&newPool.pool, func() interface{} {
		var b bytes.Buffer

		// the buffer pool will eventually home just
		// KeepAlive size buffers, so we just as well
		// start from that
		b.Grow(buf_size)
		return &b
	})
	newPool.buf_size = buf_size
	newPool.max_size = buf_size * 2

	return newPool
}

func (bp *syncPoolBufPool) GetBuffer() *bytes.Buffer {
	return bp.pool.Get().(*bytes.Buffer)
}

func (bp *syncPoolBufPool) PutBuffer(b *bytes.Buffer) {
	if b.Len() < bp.max_size {
		b.Reset()
		bp.pool.Put(b)
	}
}

func (bp *syncPoolBufPool) SetBufferCapacity(s int) {
	bp.buf_size = s
	bp.max_size = s * 2
}

func (bp *syncPoolBufPool) BufferCapacity() int {
	return bp.buf_size
}
