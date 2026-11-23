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

// MB-75538: Actuals() must not round integers beyond 2^53 through float64
func TestSetActualsLargeInt(t *testing.T) {
	big := []int64{9007199254740993, 153264729320753985, -9007199254740993}
	items := make([]interface{}, 0, len(big)+1)
	for _, i := range big {
		items = append(items, i)
	}
	items = append(items, big[0]) // duplicate

	set := NewSet(len(items), true, false)
	set.AddAll(items)
	actuals := set.Actuals()
	if len(actuals) != len(big) {
		t.Fatalf("expected %d distinct values, got %d: %v", len(big), len(actuals), actuals)
	}

	seen := make(map[int64]bool, len(big))
	for _, a := range actuals {
		i, ok := a.(int64)
		if !ok {
			t.Fatalf("expected int64, got %T (%v)", a, a)
		}
		seen[i] = true
	}
	for _, i := range big {
		if !seen[i] {
			t.Errorf("value %d lost; actuals %v", i, actuals)
		}
	}
}

// MB-75538: MultiSet.Actuals() must not round integers beyond 2^53 either
func TestMultiSetActualsLargeInt(t *testing.T) {
	big := int64(9007199254740993)
	mset := NewMultiSet(4, true, false)
	mset.AddAll([]interface{}{big, big, int64(1)})
	if c := mset.Count(NewValue(big)); c != 2 {
		t.Errorf("expected count 2 for %d, got %d", big, c)
	}
	if mset.Has(NewValue(int64(9007199254740992))) {
		t.Errorf("9007199254740992 must not match %d", big)
	}

	found := false
	for _, a := range mset.Actuals() {
		switch a := a.(type) {
		case int64:
			found = found || a == big
		default:
			t.Fatalf("expected int64, got %T (%v)", a, a)
		}
	}
	if !found {
		t.Errorf("value %d lost; actuals %v", big, mset.Actuals())
	}
}
