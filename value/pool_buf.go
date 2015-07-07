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
	"bytes"
	"sync"
)

const _BUF_SIZE = 512

var _BUF_POOL = &sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, _BUF_SIZE))
	},
}

func allocateBuf() *bytes.Buffer {
	//TODO: FIXME: return _BUF_POOL.Get().(*bytes.Buffer)
	return bytes.NewBuffer(make([]byte, 0, _BUF_SIZE))
}

func releaseBuf(b *bytes.Buffer) {
	/*
		if b == nil {
			return
		}

		b.Reset()
		_BUF_POOL.Put(b)
	*/
}
