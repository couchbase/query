//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package datastore

import (
	"github.com/couchbase/query/util"
)

type IndexPool struct {
	pool util.FastPool
	size int
}

func NewIndexPool(size int) *IndexPool {
	rv := &IndexPool{
		size: size,
	}
	util.NewFastPool(&rv.pool, func() interface{} {
		return make([]Index, 0, size)
	})

	return rv
}

func (this IndexPool) Get() []Index {
	return this.pool.Get().([]Index)
}

func (this IndexPool) Put(s []Index) {
	if cap(s) != this.size {
		return
	}

	for i := range s {
		s[i] = nil
	}
	this.pool.Put(s[0:0])
}
