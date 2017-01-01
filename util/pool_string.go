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

type StringPool struct {
	pool *sync.Pool
	size int
}

func NewStringPool(size int) *StringPool {
	rv := &StringPool{
		pool: &sync.Pool{
			New: func() interface{} {
				return make([]string, 0, size)
			},
		},
		size: size,
	}

	return rv
}

func (this *StringPool) Get() []string {
	return this.pool.Get().([]string)
}

func (this *StringPool) GetCapped(capacity int) []string {
	if capacity > this.size {
		return make([]string, 0, capacity)
	} else {
		return this.Get()
	}
}

func (this *StringPool) GetSized(length int) []string {
	if length > this.size {
		return make([]string, length)
	}

	rv := this.Get()
	rv = rv[0:length]
	for i := 0; i < length; i++ {
		rv[i] = ""
	}

	return rv
}

func (this *StringPool) Put(s []string) {
	if cap(s) != this.size {
		return
	}

	this.pool.Put(s[0:0])
}
