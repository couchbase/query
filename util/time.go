//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package util

import (
	"math"
	"sync"
	"sync/atomic"
	"time"
	_ "unsafe" // required to use //go:linkname
)

const DEFAULT_FORMAT = "2006-01-02T15:04:05.999Z07:00"

var ZERO_DURATION_STR = time.Duration(0).String()

// util.Time is the monotonic clock; base is the offset that turns it into wall clock time.
// The offset goes stale when the system clock is changed, until the next resync
var base int64

const (
	_RESYNC_INTERVAL = time.Second      // StartTimeResync() frequency
	_RESYNC_MIN_SKEW = time.Millisecond // smallest offset change ResyncTime() applies
	_RESYNC_SAMPLES  = 3                // clock readings ResyncTime() takes
)

var resyncOnce sync.Once

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

func (this Time) ToTime() time.Time {
	un := int64(this.UnixNano())
	return time.Unix(un/1e9, un%1e9)
}

// ResyncTime refreshes the wall clock offset when it has moved by more than the reading error plus
// _RESYNC_MIN_SKEW, so that the offset only moves for an actual clock change and not because of
// jitter between clock reads
func ResyncTime() {
	old := atomic.LoadInt64(&base)

	// take the wall clock reading that was least delayed, e.g. by preemption, between two
	// monotonic clock readings
	var b int64
	gap := int64(math.MaxInt64)
	for i := 0; i < _RESYNC_SAMPLES; i++ {
		m1 := nanotime()
		w := time.Now().UnixNano()
		m2 := nanotime()
		if m2-m1 < gap {
			b = w - (m1+m2)/2
			gap = m2 - m1
		}
	}

	if applyOffset(old, b, gap) {

		// a concurrent resync has already stored a newer offset
		atomic.CompareAndSwapInt64(&base, old, b)
	}
}

// applyOffset reports whether ResyncTime() replaces offset old with offset b, read with gap
// nanoseconds between its monotonic clock readings.
// b is off by at most gap/2, so a change of more than gap + _RESYNC_MIN_SKEW is a clock change and
// not jitter. Slow readings, e.g. on a CPU throttled host, only raise the threshold: they cannot
// hold back a clock change of seconds or more. The first resync always stores an offset
func applyOffset(old, b, gap int64) bool {
	if old == 0 {
		return true
	}
	d := b - old
	if d < 0 {
		d = -d
	}
	return d > gap+int64(_RESYNC_MIN_SKEW)
}

// StartTimeResync keeps the wall clock offset within _RESYNC_INTERVAL of a system clock change,
// so that deadlines computed from util.Now() don't go stale. Only the first call starts the resync
func StartTimeResync() {
	resyncOnce.Do(func() {
		startTimeResync(_RESYNC_INTERVAL)
	})
}

func startTimeResync(interval time.Duration) (stop func()) {
	done := make(chan struct{})
	exited := make(chan struct{})
	go func() {
		defer close(exited)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ResyncTime()
			case <-done:
				return
			}
		}
	}()
	return func() {
		close(done)
		<-exited
	}
}

// SkewTimeForTest shifts util.Time's wall clock time by d, as if the system clock had moved by -d
// since the last resync.
// For tests only: it is exported for the tests of other packages. Never call it in production code,
// it makes util.Now() wrong until the next resync
func SkewTimeForTest(d time.Duration) {
	atomic.AddInt64(&base, int64(d))
}
