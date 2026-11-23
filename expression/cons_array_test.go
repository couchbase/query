//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"testing"

	"github.com/couchbase/query/value"
)

// MB-75538: IN-list de-dup/sort must keep int64 precision beyond 2^53
func TestSortValArrLargeInt(t *testing.T) {
	vals := []any{int64(9007199254740993), int64(1), int64(9007199254740993), "a", 2.5}
	out := SortValArr(vals)
	if len(out) != 4 {
		t.Fatalf("expected 4 values, got %d: %v", len(out), out)
	}

	expected := value.NewValue([]any{int64(1), 2.5, int64(9007199254740993), "a"})
	if !value.NewValue(out).Equals(expected).Truth() {
		t.Errorf("expected %v, got %v", expected, value.NewValue(out))
	}

	c := NewConstant(out[2])
	if c.String() != "9007199254740993" {
		t.Errorf("span constant rounded: %s", c.String())
	}
}
