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
	"crypto/rand"
	"fmt"
	"io"
	"sync"
)

var uuidPool *sync.Pool

const _UUID_SIZE = 16

func init() {
	uuidPool = &sync.Pool{
		New: func() interface{} {
			b := make([]byte, _UUID_SIZE, _UUID_SIZE)
			return b
		},
	}
}

// UUID generates a random UUID according to RFC 4122
func UUID() (string, error) {
	uuid := uuidPool.Get().([]byte)
	defer uuidPool.Put(uuid)

	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}
