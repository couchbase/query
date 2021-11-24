//  Copyright 2018-Present Couchbase, Inc.
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

func TestUUID(t *testing.T) {
	u1, _ := UUIDV4()
	u2, _ := UUIDV4()

	v5u1, _ := UUIDV5(u1, u2)
	v5u2, _ := UUIDV5(u1, u2)

	if v5u1 != v5u2 {
		t.Errorf("Expected %v twice, got %v instead", v5u1, v5u2)
	}
}
