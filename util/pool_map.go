//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package util

// MapPool is a generic pool of maps, replacing the hand-written per-type map
// pools. Put() uses the clear() builtin, which both empties the map and zeroes
// the values, so pooled maps never retain references to recycled objects.
type MapPool[K comparable, V any] struct {
	pool FastPool
	size int
}

func NewMapPool[K comparable, V any](size int) *MapPool[K, V] {
	rv := &MapPool[K, V]{
		size: size,
	}
	NewFastPool(&rv.pool, func() any {
		return make(map[K]V, rv.size)
	})

	return rv
}

func (this *MapPool[K, V]) Get() map[K]V {
	return this.pool.Get().(map[K]V)
}

func (this *MapPool[K, V]) GetCapped(capacity int) map[K]V {
	if capacity <= this.size {
		return this.Get()
	} else {
		return make(map[K]V, capacity)
	}
}

func (this *MapPool[K, V]) Put(s map[K]V) {
	if s == nil || len(s) > this.size {
		return
	}

	clear(s)

	this.pool.Put(s)
}

func (this *MapPool[K, V]) Size() int {
	return this.size
}
