//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package util

// SlicePool is a generic pool of slices, replacing the hand-written per-type
// slice pools. Put() uses the clear() builtin to zero the elements, so pooled
// slices never retain references to recycled objects.
type SlicePool[T any] struct {
	pool FastPool
	size int
}

func NewSlicePool[T any](size int) *SlicePool[T] {
	rv := &SlicePool[T]{
		size: size,
	}
	NewFastPool(&rv.pool, func() any {
		return make([]T, 0, size)
	})

	return rv
}

func (this *SlicePool[T]) Get() []T {
	return this.pool.Get().([]T)
}

func (this *SlicePool[T]) GetCapped(capacity int) []T {
	if capacity > this.size {
		return make([]T, 0, capacity)
	} else {
		return this.Get()
	}
}

func (this *SlicePool[T]) GetSized(length int) []T {
	if length > this.size {
		return make([]T, length)
	}

	rv := this.Get()
	rv = rv[0:length]
	clear(rv)

	return rv
}

func (this *SlicePool[T]) Put(s []T) {
	if cap(s) != this.size {
		return
	}

	// Clear the whole backing array, not just len(s): callers commonly do
	// s := pool.Get(); defer pool.Put(s) and then append to s, so the
	// deferred Put sees a len-0 header while the array is populated.
	clear(s[:cap(s)])
	this.pool.Put(s[0:0])
}

func (this *SlicePool[T]) Size() int {
	return this.size
}
