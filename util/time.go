//  Copyright (c) 2019 Couchbase, Inc.
//
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package util

import (
	"time"
	_ "unsafe" // required to use //go:linkname
)

type Time int64

//go:noescape
//go:linkname nanotime runtime.nanotime
func nanotime() int64

func Now() Time {
	return Time(nanotime())
}

func Since(t Time) time.Duration {
	return time.Duration(Time(nanotime()) - t)
}

func (this Time) Sub(t Time) time.Duration {
	return time.Duration(this - t)
}
