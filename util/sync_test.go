//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package util

import (
	"testing"
	"time"

	atomic "github.com/couchbase/go-couchbase/platform"
)

// The drainer runs beside Get and Put, so these tests also check, under
// go test -race, that it reads the pool counters safely.

const _TEST_DRAIN_INTERVAL = time.Second

func newDrainedTestPool(t *testing.T) *FastPool {
	t.Helper()
	p := &FastPool{}
	NewFastPool(p, func() any { return new(int) })
	if err := p.Drain(20, 40, _TEST_DRAIN_INTERVAL); err != nil {
		t.Fatalf("Drain: %v", err)
	}
	t.Cleanup(p.Close)

	// fill the pool above the high water mark
	for i := 0; i < _POOL_SIZE; i++ {
		p.Put(new(int))
	}
	if n := atomic.LoadInt32(&p.useCount); n <= p.high {
		t.Fatalf("expected more than %d entries, got %d", p.high, n)
	}
	return p
}

// An unused pool above the high water mark is trimmed to the low water mark.
func TestFastPoolDrainIdle(t *testing.T) {
	t.Parallel()
	p := newDrainedTestPool(t)

	deadline := time.Now().Add(5 * _TEST_DRAIN_INTERVAL)
	for atomic.LoadInt32(&p.useCount) > p.low {
		if time.Now().After(deadline) {
			t.Fatalf("pool not drained: %d entries, low water mark %d",
				atomic.LoadInt32(&p.useCount), p.low)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// A pool that sees Get calls between drainer checks is left alone.
func TestFastPoolDrainInUse(t *testing.T) {
	t.Parallel()
	p := newDrainedTestPool(t)

	end := time.Now().Add(2*_TEST_DRAIN_INTERVAL + _TEST_DRAIN_INTERVAL/4)
	for time.Now().Before(end) {
		p.Put(p.Get())
		time.Sleep(50 * time.Millisecond)
	}

	if n := atomic.LoadInt32(&p.useCount); n <= p.high {
		t.Fatalf("pool in use was drained: %d entries, high water mark %d", n, p.high)
	}
}
