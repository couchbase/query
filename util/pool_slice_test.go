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

const _TEST_POOL_SIZE = 4

func assertCleared(t *testing.T, s []*int) {
	t.Helper()
	if len(s) != 0 {
		t.Fatalf("expected len 0, got %d", len(s))
	}
	for i, p := range s[:cap(s)] {
		if p != nil {
			t.Fatalf("element %d of pooled backing array not cleared", i)
		}
	}
}

// Put() must clear the full backing array even when given a len-0 header,
// as happens with s := pool.Get(); defer pool.Put(s); s = append(s, ...)
func TestSlicePoolDeferredPut(t *testing.T) {
	pool := NewSlicePool[*int](_TEST_POOL_SIZE)

	func() {
		s := pool.Get()
		defer pool.Put(s)
		for i := 0; i < _TEST_POOL_SIZE; i++ {
			v := i
			s = append(s, &v)
		}
	}()

	assertCleared(t, pool.Get())
}

// Appending past capacity reallocates; the original array still goes back to
// the pool via the deferred Put and must be cleared.
func TestSlicePoolDeferredPutAfterGrowth(t *testing.T) {
	pool := NewSlicePool[*int](_TEST_POOL_SIZE)

	func() {
		s := pool.Get()
		defer pool.Put(s)
		for i := 0; i < 2*_TEST_POOL_SIZE; i++ {
			v := i
			s = append(s, &v)
		}
	}()

	assertCleared(t, pool.Get())
}

func TestSlicePoolPut(t *testing.T) {
	pool := NewSlicePool[*int](_TEST_POOL_SIZE)

	s := pool.Get()
	for i := 0; i < _TEST_POOL_SIZE; i++ {
		v := i
		s = append(s, &v)
	}
	pool.Put(s)

	assertCleared(t, pool.Get())
}

// Slices of the wrong capacity must not enter the pool.
func TestSlicePoolPutWrongCapacity(t *testing.T) {
	pool := NewSlicePool[*int](_TEST_POOL_SIZE)

	pool.Put(make([]*int, 0, 2*_TEST_POOL_SIZE))
	if s := pool.Get(); cap(s) != _TEST_POOL_SIZE {
		t.Fatalf("expected cap %d, got %d", _TEST_POOL_SIZE, cap(s))
	}
}

func TestSlicePoolGetSized(t *testing.T) {
	pool := NewSlicePool[*int](_TEST_POOL_SIZE)

	s := pool.GetSized(_TEST_POOL_SIZE)
	if len(s) != _TEST_POOL_SIZE {
		t.Fatalf("expected len %d, got %d", _TEST_POOL_SIZE, len(s))
	}
	for i, p := range s {
		if p != nil {
			t.Fatalf("element %d not zeroed", i)
		}
	}

	if s = pool.GetSized(2 * _TEST_POOL_SIZE); len(s) != 2*_TEST_POOL_SIZE {
		t.Fatalf("expected len %d, got %d", 2*_TEST_POOL_SIZE, len(s))
	}
}

func TestSlicePoolGetCapped(t *testing.T) {
	pool := NewSlicePool[*int](_TEST_POOL_SIZE)

	if s := pool.GetCapped(_TEST_POOL_SIZE / 2); cap(s) != _TEST_POOL_SIZE {
		t.Fatalf("expected pooled cap %d, got %d", _TEST_POOL_SIZE, cap(s))
	}
	if s := pool.GetCapped(2 * _TEST_POOL_SIZE); cap(s) != 2*_TEST_POOL_SIZE {
		t.Fatalf("expected cap %d, got %d", 2*_TEST_POOL_SIZE, cap(s))
	}
}
