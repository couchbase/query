//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package util

import (
	"sync/atomic"
	"testing"
	"time"
)

const clockJump = 48 * time.Hour

// how far util.Now()'s wall clock time is from the system clock
func clockSkew() time.Duration {
	return time.Duration(Now().UnixNano() - time.Now().UnixNano())
}

func TestResyncTime(t *testing.T) {
	defer ResyncTime()

	for _, d := range []time.Duration{-clockJump, clockJump} {
		ResyncTime()
		SkewTimeForTest(d)
		if s := clockSkew(); (s - d).Abs() > time.Second {
			t.Fatalf("expected skew %v, got %v", d, s)
		}
		ResyncTime()
		if s := clockSkew().Abs(); s > time.Second {
			t.Fatalf("skew %v after resync of %v", s, d)
		}
	}
}

// the first resync stores an offset even without a reading delayed by less than _RESYNC_MIN_SKEW
func TestResyncTimeUnset(t *testing.T) {
	defer ResyncTime()

	atomic.StoreInt64(&base, 0)
	ResyncTime()
	if s := clockSkew().Abs(); s > time.Second {
		t.Fatalf("skew %v after first resync", s)
	}
}

func TestApplyOffset(t *testing.T) {
	ms := int64(time.Millisecond)
	jump := int64(clockJump)
	for _, c := range []struct {
		name        string
		old, b, gap int64
		expect      bool
	}{
		{"unset", 0, 5, 50 * ms, true},
		{"jitter", jump, jump + ms/2, ms / 10, false},
		{"small change", jump, jump + 2*ms, ms / 10, true},
		{"small change, slow reading", jump, jump + 2*ms, 5 * ms, false},
		{"clock forward, slow readings", jump, 2 * jump, 50 * ms, true},
		{"clock back, slow readings", 2 * jump, jump, 50 * ms, true},
	} {
		if got := applyOffset(c.old, c.b, c.gap); got != c.expect {
			t.Errorf("%s: expected %v, got %v", c.name, c.expect, got)
		}
	}
}

func TestResyncTimeStable(t *testing.T) {
	ResyncTime()
	b := atomic.LoadInt64(&base)
	for i := 0; i < 1000; i++ {
		ResyncTime()
	}
	if atomic.LoadInt64(&base) != b {
		t.Fatalf("offset moved without a clock change")
	}
}

func TestTimeResyncLoop(t *testing.T) {
	defer ResyncTime()

	interval := 10 * time.Millisecond
	stop := startTimeResync(interval)
	defer stop()

	for _, d := range []time.Duration{-clockJump, clockJump} {
		SkewTimeForTest(d)
		deadline := time.Now().Add(time.Second)
		for clockSkew().Abs() > time.Second {
			if time.Now().After(deadline) {
				t.Fatalf("skew of %v not resynced within %v", d, time.Second)
			}
			time.Sleep(interval)
		}
	}
}
