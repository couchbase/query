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
)

func assertQueueCleared(t *testing.T, q *Queue) {
	t.Helper()
	if q.Size() != 0 {
		t.Fatalf("expected size 0, got %d", q.Size())
	}
	for i, n := range q.nodes {
		if n != nil {
			t.Fatalf("node %d not cleared: %v", i, n)
		}
	}
}

func TestQueueClear(t *testing.T) {
	cases := []struct {
		name   string
		fill   int // added first
		drain  int // then removed
		refill int // then added again, wrapping around
	}{
		{"empty", 0, 0, 0},
		{"partial", 2, 0, 0},
		{"full", _TEST_POOL_SIZE, 0, 0},
		{"full wrapped", _TEST_POOL_SIZE, 2, 2},
		{"partial wrapped", _TEST_POOL_SIZE, 3, 2},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			q := NewQueue(_TEST_POOL_SIZE)
			for i := 0; i < c.fill; i++ {
				q.Add(i)
			}
			for i := 0; i < c.drain; i++ {
				q.Remove()
			}
			for i := 0; i < c.refill; i++ {
				q.Add(c.fill + i)
			}
			if q.Capacity() != _TEST_POOL_SIZE {
				t.Fatalf("queue grew to %d, test setup is wrong", q.Capacity())
			}

			q.Clear()
			assertQueueCleared(t, q)

			// the queue must still work in FIFO order after Clear
			for i := 0; i < _TEST_POOL_SIZE; i++ {
				q.Add(i)
			}
			for i := 0; i < _TEST_POOL_SIZE; i++ {
				if n := q.Remove(); n != i {
					t.Fatalf("expected %d, got %v", i, n)
				}
			}
		})
	}
}

// A full queue put back in the pool must not keep its entries.
func TestQueuePoolPutFull(t *testing.T) {
	pool := NewQueuePool(_TEST_POOL_SIZE)

	q := pool.Get()
	for i := 0; i < _TEST_POOL_SIZE; i++ {
		q.Add(i)
	}
	pool.Put(q)

	assertQueueCleared(t, q)
}
