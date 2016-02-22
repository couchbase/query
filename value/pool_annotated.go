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
	"sync"
)

type AnnotatedPool struct {
	pool *sync.Pool
	size int
}

func NewAnnotatedPool(size int) *AnnotatedPool {
	rv := &AnnotatedPool{
		pool: &sync.Pool{
			New: func() interface{} {
				return make(AnnotatedValues, 0, size)
			},
		},
		size: size,
	}

	return rv
}

func (this *AnnotatedPool) Get() AnnotatedValues {
	return this.pool.Get().(AnnotatedValues)
}

func (this *AnnotatedPool) Put(s AnnotatedValues) {
	if cap(s) < this.size {
		return
	}

	this.pool.Put(s[0:0])
}

func (this *AnnotatedPool) Size() int {
	return this.size
}
