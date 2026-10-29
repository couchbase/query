//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of the
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package extparams

import (
	"testing"
)

func TestGetValTypeIntRange(t *testing.T) {
	for _, c := range []struct {
		f    float64
		want string
	}{
		{5, "int"},
		{-0x1p63, "int"},
		{0x1p63, "float64"}, // arm64 saturates int64(2^63) to MaxInt64
		{1e19, "float64"},
		{1.5, "float64"},
	} {
		if got := getValType(c.f); got != c.want {
			t.Errorf("getValType(%g) = %q, want %q", c.f, got, c.want)
		}
	}
}
