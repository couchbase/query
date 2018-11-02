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
	"crypto/sha1"
	"fmt"
	"io"
	"sync"

	atomic "github.com/couchbase/go-couchbase/platform"
)

const _UUID_SIZE = 16

type randomBytesBuffer struct {
	switchHandler sync.WaitGroup
	switchWaiter  sync.WaitGroup
	currentBuffer []byte
	nextBuffer    []byte
	currentIndex  uint32
	err           error
}

var randomBytes randomBytesBuffer

const _RANDOM_SIZE = 32768 * _UUID_SIZE
const _FIRE_THRESHOLD = _RANDOM_SIZE / 2
const _RANDOM_LOCKER = _RANDOM_SIZE + _UUID_SIZE

func init() {
	randomBytes.currentBuffer = make([]byte, _RANDOM_SIZE)
	randomBytes.nextBuffer = make([]byte, _RANDOM_SIZE)
	randomBytes.switchHandler.Add(1)
	randomBytes.switchWaiter.Add(1)
	getNextBuffer()
	tmp := randomBytes.currentBuffer
	randomBytes.currentBuffer = randomBytes.nextBuffer
	randomBytes.nextBuffer = tmp
}

func readFull(bytes []byte) error {
	for {

		// copy pointer so that the structure can be changed
		buffer := randomBytes.currentBuffer

		// get next position
		index := atomic.AddUint32(&randomBytes.currentIndex, _UUID_SIZE)

		// we are close to needing the next buffer
		if index == _FIRE_THRESHOLD {
			randomBytes.switchHandler.Add(1)
			randomBytes.switchWaiter.Add(1)
			go getNextBuffer()
		}

		// we are in luck
		if index <= _RANDOM_SIZE {
			copy(bytes, buffer[index-_UUID_SIZE:index])
			return nil
		}

		// out of space - slow path
		// first reader waiting does the dirty work
		if index == _RANDOM_LOCKER {

			// wait for the asynchronous read
			randomBytes.switchHandler.Wait()

			// it didn't work
			if randomBytes.err != nil {

				// wake the waiters (if extra readers come along they will
				// get an error anyway)
				randomBytes.switchWaiter.Done()

				// try again: block everyone
				randomBytes.switchHandler.Add(1)
				randomBytes.switchWaiter.Add(1)

				// set up another first reader
				atomic.StoreUint32(&randomBytes.currentIndex, _RANDOM_SIZE)

				go getNextBuffer()
				return randomBytes.err
			}

			// switch buffer
			tmp := randomBytes.currentBuffer
			randomBytes.currentBuffer = randomBytes.nextBuffer
			randomBytes.nextBuffer = tmp

			// get our random data
			copy(bytes, randomBytes.currentBuffer[0:_UUID_SIZE])

			// reset the index and wake up the others
			atomic.StoreUint32(&randomBytes.currentIndex, _UUID_SIZE)
			randomBytes.switchWaiter.Done()
			return nil
		} else {

			// everyone else takes advantage
			randomBytes.switchHandler.Wait()
			if randomBytes.err != nil {
				return randomBytes.err
			}

			// try again
			continue
		}
	}
}

func getNextBuffer() {
	n, err := io.ReadFull(rand.Reader, randomBytes.nextBuffer)
	if n != _RANDOM_SIZE {
		randomBytes.err = fmt.Errorf("random reader only provide %v bytes", n)
	} else {
		randomBytes.err = err
	}
	randomBytes.switchHandler.Done()
}

// UUID generates a random UUID according to RFC 4122
func UUID() (string, error) {
	var bytes [_UUID_SIZE]byte
	uuid := bytes[0:_UUID_SIZE:_UUID_SIZE]

	err := readFull(uuid)
	if err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

// UUIDV5 generates a V5 UUID based on data according to RFC 4122
func UUIDV5(space, data string) (string, error) {
	var bytes [_UUID_SIZE]byte
	uuid := bytes[0:_UUID_SIZE:_UUID_SIZE]

	h := sha1.New()
	h.Write([]byte(space))
	h.Write([]byte(data))
	s := h.Sum(nil)
	copy(uuid, s)

	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 5 (sha1); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x50
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}
