//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.
//
//go:build enterprise

package prepareds

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// MB-75478: the text saved with an ad hoc plan does not cut a (UTF-8) character
func TestTextPrefix(t *testing.T) {
	short := "SELECT 1"
	if textPrefix(short) != short {
		t.Errorf("short text changed: %q", textPrefix(short))
	}
	exact := strings.Repeat("x", _TEXT_SIZE)
	if textPrefix(exact) != exact {
		t.Errorf("text of %d bytes changed", _TEXT_SIZE)
	}

	for _, char := range []string{"é", "€", "😀"} { // 2, 3 and 4 bytes
		for pad := 0; pad < len(char); pad++ {
			// the character straddles byte _TEXT_SIZE
			text := strings.Repeat("x", _TEXT_SIZE-len(char)+1+pad) + char + " FROM b"
			prefix := textPrefix(text)
			cut := strings.TrimSuffix(prefix, "...")
			if !strings.HasSuffix(prefix, "...") || !utf8.ValidString(prefix) || len(cut) > _TEXT_SIZE ||
				!strings.HasPrefix(text, cut) {
				t.Errorf("%q (pad %d): prefix %q", char, pad, prefix)
			}
		}
	}
}
