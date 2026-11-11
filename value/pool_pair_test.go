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

// A deferred Put evaluates its argument while the slice is still empty, and the
// caller fills the backing array afterwards. Put must still clear every slot so
// the pool does not keep the batch's keys and documents reachable.
func TestPairPoolPutClearsBackingArray(t *testing.T) {
	const size = 8
	pool := NewPairPool(size)

	pairs := pool.Get()
	empty := pairs // what `defer pool.Put(pairs)` would capture
	pairs = pairs[0:size]
	for i := range pairs {
		pairs[i] = Pair{Name: "k", Value: NewValue(map[string]interface{}{"doc": i})}
	}

	pool.Put(empty)

	for i, p := range empty[:cap(empty)] {
		if p.Name != "" || p.Value != nil || p.Options != nil {
			t.Fatalf("slot %d not cleared after Put: %+v", i, p)
		}
	}
}
