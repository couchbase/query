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

type PairPool struct {
	pool *sync.Pool
	size int
}

func NewPairPool(size int) *PairPool {
	rv := &PairPool{
		pool: &sync.Pool{
			New: func() interface{} {
				return make([]Pair, 0, size)
			},
		},
		size: size,
	}

	return rv
}

func (this *PairPool) Get() []Pair {
	return this.pool.Get().([]Pair)
}

func (this *PairPool) Put(s []Pair) {
	if cap(s) < this.size || cap(s) > 2*this.size {
		return
	}

	this.pool.Put(s[0:0])
}
