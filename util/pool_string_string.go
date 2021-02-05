//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package util

type StringStringPool struct {
	pool FastPool
	size int
}

func NewStringStringPool(size int) *StringStringPool {
	rv := &StringStringPool{
		size: size,
	}
	NewFastPool(&rv.pool, func() interface{} {
		return make(map[string]string, rv.size)
	})

	return rv
}

func (this *StringStringPool) Get() map[string]string {
	return this.pool.Get().(map[string]string)
}

func (this *StringStringPool) Put(s map[string]string) {
	if s == nil || len(s) > this.size {
		return
	}

	for k, _ := range s {
		s[k] = ""
		delete(s, k)
	}

	this.pool.Put(s)
}
