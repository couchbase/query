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

type BoolPool struct {
	pool *sync.Pool
	size int
}

func NewBoolPool(size int) *BoolPool {
	rv := &BoolPool{
		pool: &sync.Pool{
			New: func() interface{} {
				return make([]bool, 0, size)
			},
		},
		size: size,
	}

	return rv
}

func (this *BoolPool) Get() []bool {
	return this.pool.Get().([]bool)
}

func (this *BoolPool) GetCapped(capacity int) []bool {
	if capacity > this.size {
		return make([]bool, 0, capacity)
	} else {
		return this.Get()
	}
}

func (this *BoolPool) GetSized(length int) []bool {
	if length > this.size {
		return make([]bool, length)
	}

	rv := this.Get()
	rv = rv[0:length]
	for i := 0; i < length; i++ {
		rv[i] = false
	}

	return rv
}

func (this *BoolPool) Put(s []bool) {
	if cap(s) != this.size {
		return
	}

	this.pool.Put(s[0:0])
}
