//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package couchbase

import (
	goerrors "errors"
	"fmt"
	"testing"

	"github.com/couchbase/gomemcached"
)

// a combined bulk get error is ignored as not found only if every error in it is
func TestIsNotFoundErrorCombined(t *testing.T) {
	notFound := &gomemcached.MCResponse{Status: gomemcached.KEY_ENOENT}
	wrapped := fmt.Errorf("get failed: KEY_ENOENT")
	failed := fmt.Errorf("bulk get of v-bucket 3 failed: boom")
	for _, c := range []struct {
		err error
		exp bool
	}{
		{notFound, true},
		{wrapped, true},
		{failed, false},
		{goerrors.Join(notFound, wrapped), true},
		{goerrors.Join(notFound, failed), false},
		{goerrors.Join(wrapped, failed), false},
		{goerrors.Join(failed, notFound), false},
	} {
		if got := isNotFoundError(c.err); got != c.exp {
			t.Errorf("isNotFoundError(%q) = %v, expected %v", c.err, got, c.exp)
		}
	}
}
