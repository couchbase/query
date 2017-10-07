//  Copyright (c) 2017 Couchbase, Inc.
//
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package util

// We implement here sync types that in sync don't do exactly what we want.
// Our implementation tends to be leaner too.

import (
	atomic "github.com/couchbase/go-couchbase/platform"
)

// Once is an object that will perform exactly one action.
type Once struct {
	done uint32
}

// Do calls the function f if and only if Do is being called for the
// first time for this instance of Once. In other words, given
// 	var once Once
// if once.Do(f) is called multiple times, only the first call will invoke f,
// even if f has a different value in each invocation. A new instance of
// Once is required for each function to execute.
//
// Do is intended for initialization that must be run exactly once. Since f
// is niladic, it may be necessary to use a function literal to capture the
// arguments to a function to be invoked by Do:
// 	config.once.Do(func() { config.init(filename) })
//
// Because no call to Do returns until the one call to f returns, if f causes
// Do to be called, it will deadlock.
//
// If f panics, Do considers it to have returned; future calls of Do return
// without calling f.
//
// Our Once type can be reset
func (o *Once) Do(f func()) {
	if atomic.LoadUint32(&o.done) > 0 {
		return
	}

	// Slow-path.
	if atomic.AddUint32(&o.done, 1) == 1 {
		f()
	}
}

func (o *Once) Reset() {
	atomic.StoreUint32(&o.done, 0)
}
