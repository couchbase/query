//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package couchbase

import (
	"testing"
	"time"

	"github.com/couchbase/query/util"
)

const (
	clockJump  = 48 * time.Hour
	reqTimeout = time.Minute
)

// util.Now() is ahead of reqDeadline's wall clock time: either the clock moved back after the last
// resync, or it moved forward after the request started and util.Now() has since been resynced
func TestDeadlineAcrossClockChange(t *testing.T) {
	defer util.ResyncTime()

	for _, kvTimeout := range []time.Duration{_NO_TIMEOUT, time.Second, 2 * reqTimeout} {
		util.ResyncTime()
		reqDeadline := time.Now().Add(reqTimeout)
		util.SkewTimeForTest(clockJump)

		dl, err := getDeadline(reqDeadline, kvTimeout, 5*time.Second)
		if err != nil {
			t.Fatalf("kv timeout %v: unexpected error for unexpired request deadline: %v", kvTimeout, err)
		}

		// the deadline is the earlier of the kv timeout and what is left of the request timeout,
		// on the current wall clock
		limit := reqTimeout
		if kvTimeout > _NO_TIMEOUT && kvTimeout < limit {
			limit = kvTimeout
		}
		left := time.Until(dl)
		if left <= 0 || left > limit+deadlineGranularity {
			t.Fatalf("kv timeout %v: expected deadline within %v, got %v", kvTimeout, limit, left)
		}
	}

	// a request deadline that has really passed still expires
	util.ResyncTime()
	if _, err := getDeadline(time.Now().Add(-time.Second), _NO_TIMEOUT, 5*time.Second); err == nil {
		t.Fatalf("expected expired request deadline")
	}
}

func TestDeadline(t *testing.T) {
	util.ResyncTime()

	check := func(name string, dl time.Time, limit time.Duration) {
		left := time.Until(dl)
		if left <= 0 || left > limit+deadlineGranularity {
			t.Fatalf("%s: expected deadline within %v, got %v", name, limit, left)
		}
	}

	dl, _ := getDeadline(noDeadline, _NO_TIMEOUT, 5*time.Second)
	check("default", dl, 5*time.Second)
	if dl, _ = getDeadline(noDeadline, _NO_TIMEOUT, 0); !dl.IsZero() {
		t.Fatalf("expected no deadline, got %v", dl)
	}

	dl, _ = getDeadline(noDeadline, time.Second, 5*time.Second)
	check("kv timeout", dl, time.Second)

	reqDeadline := time.Now().Add(reqTimeout)
	dl, _ = getDeadline(reqDeadline, _NO_TIMEOUT, 5*time.Second)
	check("request deadline", dl, reqTimeout)
	dl, _ = getDeadline(reqDeadline, time.Second, 5*time.Second)
	check("kv timeout before request deadline", dl, time.Second)
	dl, _ = getDeadline(reqDeadline, 2*reqTimeout, 5*time.Second)
	check("request deadline before kv timeout", dl, reqTimeout)

	// the deadline is rounded, so that repeated calls reuse it
	dl1, _ := getDeadline(noDeadline, time.Second, 5*time.Second)
	if dl1.UnixNano()%int64(deadlineGranularity) != 0 {
		t.Fatalf("deadline %v not rounded to %v", dl1, deadlineGranularity)
	}
}
