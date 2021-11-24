//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// package couchbase provides low level access to the KV store and the orchestrator
package couchbase

import (
	"testing"
)

func TestParseURL(t *testing.T) {
	tests := []struct {
		in    string
		works bool
	}{
		{"", false},
		{"http://whatever/", true},
		{"http://%/", false},
	}

	for _, test := range tests {
		got, err := ParseURL(test.in)
		switch {
		case err == nil && test.works,
			!(err == nil || test.works):
		case err == nil && !test.works:
			t.Errorf("Expected failure on %v, got %v", test.in, got)
		case test.works && err != nil:
			t.Errorf("Expected success on %v, got %v", test.in, err)
		}
	}
}
