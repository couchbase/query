//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package value

import (
	"github.com/couchbase/query/util"
)

type PairPool struct {
	pool util.FastPool
	size int
}

func NewPairPool(size int) *PairPool {
	rv := &PairPool{
		size: size,
	}
	util.NewFastPool(&rv.pool, func() interface{} {
		return make([]Pair, 0, size)
	})

	return rv
}

func (this *PairPool) Get() []Pair {
	return this.pool.Get().([]Pair)
}

func (this *PairPool) Put(s []Pair) {
	if cap(s) != this.size {
		return
	}

	for i := range s {
		s[i] = Pair{}
	}
	this.pool.Put(s[0:0])
}

func (this *PairPool) Size() int {
	return this.size
}
