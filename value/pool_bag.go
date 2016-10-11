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

type BagPool struct {
	pool      *sync.Pool
	objectCap int
}

func NewBagPool(objectCap int) *BagPool {
	rv := &BagPool{
		pool: &sync.Pool{
			New: func() interface{} {
				return NewBag(objectCap)
			},
		},
		objectCap: objectCap,
	}

	return rv
}

func (this *BagPool) Get() *Bag {
	return this.pool.Get().(*Bag)
}

func (this *BagPool) Put(s *Bag) {
	if s.DistinctLen() > 16*this.objectCap {
		return
	}

	s.Clear()
	this.pool.Put(s)
}
