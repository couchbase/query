//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// package couchbase provides low level access to the KV store and the orchestrator
package couchbase

import "testing"

func TestWriteOptionsString(t *testing.T) {
	tests := []struct {
		opts WriteOptions
		exp  string
	}{
		{Raw, "raw"},
		{AddOnly, "addonly"},
		{Persist, "persist"},
		{Indexable, "indexable"},
		{Append, "append"},
		{AddOnly | Raw, "raw|addonly"},
		{0, "0x0"},
		{Raw | AddOnly | Persist | Indexable | Append,
			"raw|addonly|persist|indexable|append"},
		{Raw | 8192, "raw|0x2000"},
	}

	for _, test := range tests {
		got := test.opts.String()
		if got != test.exp {
			t.Errorf("Expected %v, got %v", test.exp, got)
		}
	}
}
