//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package value

import (
	"testing"
)

const _TEST_ANNOTATED_POOL_SIZE = 4

func assertAnnotatedCleared(t *testing.T, s AnnotatedValues) {
	t.Helper()
	for i, v := range s {
		if v != nil {
			t.Fatalf("element %d of pooled backing array not cleared", i)
		}
	}
}

// Put() must clear the full backing array even when given a len-0 header,
// as happens with s := pool.Get(); defer pool.Put(s); s = append(s, ...)
// and with s = s[0:0]; pool.Put(s)
func TestAnnotatedPoolDeferredPut(t *testing.T) {
	pool := NewAnnotatedPool(_TEST_ANNOTATED_POOL_SIZE)

	var backing AnnotatedValues
	func() {
		s := pool.Get()
		backing = s[:cap(s)]
		defer pool.Put(s)
		for i := 0; i < _TEST_ANNOTATED_POOL_SIZE; i++ {
			s = append(s, NewAnnotatedValue(i))
		}
	}()

	assertAnnotatedCleared(t, backing)
}

// Appending past capacity reallocates; the original array still goes back to
// the pool via the deferred Put and must be cleared.
func TestAnnotatedPoolDeferredPutAfterGrowth(t *testing.T) {
	pool := NewAnnotatedPool(_TEST_ANNOTATED_POOL_SIZE)

	var backing AnnotatedValues
	func() {
		s := pool.Get()
		backing = s[:cap(s)]
		defer pool.Put(s)
		for i := 0; i < 2*_TEST_ANNOTATED_POOL_SIZE; i++ {
			s = append(s, NewAnnotatedValue(i))
		}
	}()

	assertAnnotatedCleared(t, backing)
}

// Removing an element with append(s[:c], s[c+1:]...) leaves a copy of the old
// last element just past len; Put must clear it too.
func TestAnnotatedPoolPutAfterRemove(t *testing.T) {
	pool := NewAnnotatedPool(_TEST_ANNOTATED_POOL_SIZE)

	s := pool.Get()
	backing := s[:cap(s)]
	for i := 0; i < _TEST_ANNOTATED_POOL_SIZE; i++ {
		s = append(s, NewAnnotatedValue(i))
	}
	s = append(s[:1], s[2:]...)
	pool.Put(s)

	assertAnnotatedCleared(t, backing)
}
