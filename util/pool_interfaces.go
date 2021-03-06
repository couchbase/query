//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package util

type InterfacesPool struct {
	pool FastPool
	size int
}

func NewInterfacesPool(size int) *InterfacesPool {
	rv := &InterfacesPool{
		size: size,
	}
	NewFastPool(&rv.pool, func() interface{} {
		return make([][]interface{}, 0, size)
	})

	return rv
}

func (this *InterfacesPool) Get() [][]interface{} {
	return this.pool.Get().([][]interface{})
}

func (this *InterfacesPool) GetSized(length int) [][]interface{} {
	if length > this.size {
		return make([][]interface{}, length)
	}

	rv := this.Get()
	rv = rv[0:length]
	return rv
}

func (this *InterfacesPool) Put(s [][]interface{}) {
	if cap(s) != this.size {
		return
	}

	for i := range s {
		s[i] = nil
	}
	this.pool.Put(s[0:0])
}
