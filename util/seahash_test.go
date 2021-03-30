//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package util

import (
	"testing"
)

func TestSeaHash(t *testing.T) {
	s := SeaHashSum64([]byte("to be or not to be"))

	if s != 0x1b993a826f4ae575 {
		t.Errorf("Expected 0x1b993a826f4ae575, got %x", s)
	}

	s = SeaHashSum64([]byte(""))

	if s != 0xc920ca43256fdcb9 {
		t.Errorf("Expected 0xc920ca43256fdcb9, got %x", s)
	}

	s = SeaHashSum64([]byte("couchbase"))

	if s != 0x4e5d5535342df6ef {
		t.Errorf("Expected 0x, got %x4e5d5535342df6ef", s)
	}

	s = SeaHashSum64([]byte("12345678"))

	if s != 0x79476d25d4c6dfc4 {
		t.Errorf("Expected 0x79476d25d4c6dfc4, got %x", s)
	}

	s = SeaHashSum64([]byte("Couchbase N1QL"))

	if s != 0x682b2cc0145769e3 {
		t.Errorf("Expected 0x682b2cc0145769e3, got %x", s)
	}
}
