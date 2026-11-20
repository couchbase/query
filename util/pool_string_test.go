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

func assertStringsCleared(t *testing.T, s []string) {
	t.Helper()
	for i, v := range s {
		if v != "" {
			t.Fatalf("element %d of pooled backing array not cleared: %q", i, v)
		}
	}
}

// Put() must clear the full backing array even when given a len-0 header,
// as happens with s := pool.Get(); defer pool.Put(s); s = append(s, ...)
func TestStringPoolDeferredPut(t *testing.T) {
	pool := NewStringPool(_TEST_POOL_SIZE)

	var backing []string
	func() {
		s := pool.Get()
		backing = s[:cap(s)]
		defer pool.Put(s)
		for i := 0; i < _TEST_POOL_SIZE; i++ {
			s = append(s, string(rune('a'+i)))
		}
	}()

	assertStringsCleared(t, backing)
}

// Appending past capacity reallocates; the original array still goes back to
// the pool via the deferred Put and must be cleared.
func TestStringPoolDeferredPutAfterGrowth(t *testing.T) {
	pool := NewStringPool(_TEST_POOL_SIZE)

	var backing []string
	func() {
		s := pool.Get()
		backing = s[:cap(s)]
		defer pool.Put(s)
		for i := 0; i < 2*_TEST_POOL_SIZE; i++ {
			s = append(s, string(rune('a'+i)))
		}
	}()

	assertStringsCleared(t, backing)
}

// GetSized() exposes pooled elements up to the requested length, so they must
// not carry strings from a previous user.
func TestStringPoolGetSizedAfterDeferredPut(t *testing.T) {
	pool := NewStringPool(_TEST_POOL_SIZE)

	func() {
		s := pool.Get()
		defer pool.Put(s)
		for i := 0; i < _TEST_POOL_SIZE; i++ {
			s = append(s, string(rune('a'+i)))
		}
	}()

	s := pool.GetSized(_TEST_POOL_SIZE)
	if len(s) != _TEST_POOL_SIZE {
		t.Fatalf("expected len %d, got %d", _TEST_POOL_SIZE, len(s))
	}
	assertStringsCleared(t, s)
}
