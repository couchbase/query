//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"github.com/couchbase/query/util"
)

type UnnestPool struct {
	pool util.FastPool
	size int
}

func NewUnnestPool(size int) *UnnestPool {
	rv := &UnnestPool{
		size: size,
	}
	util.NewFastPool(&rv.pool, func() interface{} {
		return make([]*Unnest, 0, size)
	})

	return rv
}

func (this *UnnestPool) Get() []*Unnest {
	return this.pool.Get().([]*Unnest)
}

func (this *UnnestPool) Put(buf []*Unnest) {
	if cap(buf) < this.size || cap(buf) > 2*this.size {
		return
	}

	for i := range buf {
		buf[i] = nil
	}
	this.pool.Put(buf[0:0])
}
