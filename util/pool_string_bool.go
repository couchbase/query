//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package util

type StringBoolPool struct {
	pool FastPool
	size int
}

func NewStringBoolPool(size int) *StringBoolPool {
	rv := &StringBoolPool{
		size: size,
	}
	NewFastPool(&rv.pool, func() interface{} {
		return make(map[string]bool, size)
	})

	return rv
}

func (this *StringBoolPool) Get() map[string]bool {
	return this.pool.Get().(map[string]bool)
}

func (this *StringBoolPool) Put(s map[string]bool) {
	if s == nil || len(s) > this.size {
		return
	}

	for k, _ := range s {
		delete(s, k)
	}

	this.pool.Put(s)
}

func (this *StringBoolPool) Size() int {
	return this.size
}
