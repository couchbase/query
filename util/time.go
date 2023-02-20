//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package util

import (
	"sync/atomic"
	"time"
	_ "unsafe" // required to use //go:linkname
)

const DEFAULT_FORMAT = "2006-01-02T15:04:05.999Z07:00"

var base int64

func init() {
	ResyncTime()
}

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

func (this Time) Add(d time.Duration) Time {
	return Time(this + Time(d))
}

func (this Time) Sub(t Time) time.Duration {
	return time.Duration(this - t)
}

func (this Time) Truncate(d time.Duration) Time {
	return this - (this % Time(d))
}

func (this Time) UnixNano() int64 {
	return int64(this) + atomic.LoadInt64(&base)
}

func ResyncTime() {
	atomic.StoreInt64(&base, time.Now().UnixNano()-nanotime())
}
