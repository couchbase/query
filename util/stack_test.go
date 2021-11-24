//  Copyright 2013-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package util

import (
	"testing"
)

func TestStack(t *testing.T) {
	s := Stack{}

	s.Push(1)
	s.Push(2)

	v2 := s.Pop()
	if v2 != 2 {
		t.Errorf("Expected 2, got %v", v2)
	}

	v1 := s.Pop()
	if v1 != 1 {
		t.Errorf("Expected 1, got %v", v1)
	}

	v0 := s.Pop()
	if v0 != nil {
		t.Errorf("Expected nil, got %v", v0)
	}
}
