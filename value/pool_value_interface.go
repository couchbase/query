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

type ValueInterfacePool struct {
	pool *sync.Pool
	size int
}

func NewValueInterfacePool(size int) *ValueInterfacePool {
	rv := &ValueInterfacePool{
		pool: &sync.Pool{
			New: func() interface{} {
				return make(map[Value]interface{}, size)
			},
		},
		size: size,
	}

	return rv
}

func (this *ValueInterfacePool) Get() map[Value]interface{} {
	return this.pool.Get().(map[Value]interface{})
}

func (this *ValueInterfacePool) Put(s map[Value]interface{}) {
	if s == nil || len(s) > this.size {
		return
	}

	for k, _ := range s {
		s[k] = nil
		delete(s, k)
	}

	this.pool.Put(s)
}
