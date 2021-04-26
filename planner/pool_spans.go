//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"sync"
)

type SargSpansPool struct {
	pool *sync.Pool
	size int
}

func NewSargSpansPool(size int) *SargSpansPool {
	rv := &SargSpansPool{
		pool: &sync.Pool{
			New: func() interface{} {
				return make([]SargSpans, 0, size)
			},
		},
		size: size,
	}

	return rv
}

func (this *SargSpansPool) Get() []SargSpans {
	return this.pool.Get().([]SargSpans)
}

func (this *SargSpansPool) GetSized(length int) []SargSpans {
	if length > this.size {
		return make([]SargSpans, length)
	}

	rv := this.Get()
	rv = rv[0:length]
	for i := range rv {
		rv[i] = nil
	}

	return rv
}

func (this *SargSpansPool) Put(s []SargSpans) {
	if cap(s) != this.size {
		return
	}

	for i := range s {
		s[i] = nil
	}
	this.pool.Put(s[0:0])
}

func (this *SargSpansPool) Size() int {
	return this.size
}
