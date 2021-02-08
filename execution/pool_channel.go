//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execution

import (
	"github.com/couchbase/query/util"
)

type ChannelPool struct {
	pool util.FastPool
	size int
}

func NewChannelPool(size int) *ChannelPool {
	rv := &ChannelPool{
		size: size,
	}
	util.NewFastPool(&rv.pool, func() interface{} {
		return make([]*Channel, 0, size)
	})

	return rv
}

func (this *ChannelPool) Get() []*Channel {
	return this.pool.Get().([]*Channel)
}

func (this *ChannelPool) Put(s []*Channel) {
	if cap(s) != this.size {
		return
	}

	this.pool.Put(s[0:0])
}
