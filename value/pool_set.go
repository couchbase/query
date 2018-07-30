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

type SetPool struct {
	pool      *sync.Pool
	objectCap int
	collect   bool
}

// numeric is a flag to restrict the Set to only contain numeric values(float64 and int64).
func NewSetPool(objectCap int, collect, numeric bool) *SetPool {
	rv := &SetPool{
		pool: &sync.Pool{
			New: func() interface{} {
				return NewSet(objectCap, collect, numeric)
			},
		},
		objectCap: objectCap,
		collect:   collect,
	}

	return rv
}

func (this *SetPool) Get() *Set {
	return this.pool.Get().(*Set)
}

func (this *SetPool) Put(s *Set) {
	if s.Len() > 16*this.objectCap {
		return
	}

	s.Clear()
	this.pool.Put(s)
}
