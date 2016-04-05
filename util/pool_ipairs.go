//  Copyright (c) 2016 Couchbase, Inc.
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

type IPairsPool struct {
	pool *sync.Pool
	size int
}

func NewIPairsPool(size int) *IPairsPool {
	rv := &IPairsPool{
		pool: &sync.Pool{
			New: func() interface{} {
				return make([][]IPair, 0, size)
			},
		},
		size: size,
	}

	return rv
}

func (this *IPairsPool) Get() [][]IPair {
	return this.pool.Get().([][]IPair)
}

func (this *IPairsPool) GetSized(length int) [][]IPair {
	if length > this.size {
		return make([][]IPair, length)
	}

	rv := this.Get()
	rv = rv[0:length]
	for i := 0; i < length; i++ {
		rv[i] = nil
	}

	return rv
}

func (this *IPairsPool) Put(s [][]IPair) {
	if cap(s) < this.size || cap(s) > 2*this.size {
		return
	}

	this.pool.Put(s[0:0])
}
