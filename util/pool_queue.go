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
	"sync"
)

type QueuePool struct {
	pool *sync.Pool
	size int
}

func NewQueuePool(size int) *QueuePool {
	rv := &QueuePool{
		pool: &sync.Pool{
			New: func() interface{} {
				return NewQueue(size)
			},
		},
		size: size,
	}

	return rv
}

func (this *QueuePool) Get() *Queue {
	return this.pool.Get().(*Queue)
}

func (this *QueuePool) Put(s *Queue) {
	if s.Capacity() != this.size {
		return
	}

	s.Clear()
	this.pool.Put(s)
}
