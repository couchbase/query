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
	"testing"
	"time"

	"github.com/couchbase/query/errors"
)

// a failure to write the default plan stability setting after QUERY_METADATA went missing is retried, and
// does not panic
func TestPersistDefaultPlanStabilitySetting(t *testing.T) {
	defer func(p func() errors.Error, d func() bool, i, m time.Duration) {
		persistDefaultSetting = p
		isDefaultSetting = d
		resetRetryInterval = i
		resetMaxRetryInterval = m
	}(persistDefaultSetting, isDefaultSetting, resetRetryInterval, resetMaxRetryInterval)

	resetRetryInterval = time.Millisecond
	resetMaxRetryInterval = 4 * time.Millisecond

	for _, tc := range []struct {
		name     string
		failures int // writes that fail before one succeeds
		changed  int // the setting is changed after this many writes (-1: never)
		writes   int
	}{
		{"write succeeds", 0, -1, 1},
		{"write fails, then succeeds", 5, -1, 6},
		{"setting changed while retrying", 100, 3, 3},
		{"setting changed before the write", 0, 0, 0},
	} {
		writes := 0
		failures, changed := tc.failures, tc.changed
		persistDefaultSetting = func() errors.Error {
			writes++
			if writes <= failures {
				return errors.NewSettingsMetaKVError(nil, "metakv not available")
			}
			return nil
		}
		isDefaultSetting = func() bool {
			return changed < 0 || writes < changed
		}

		done := make(chan bool)
		go func() {
			persistDefaultPlanStabilitySetting()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatalf("%s: still retrying after %d writes", tc.name, writes)
		}
		if writes != tc.writes {
			t.Errorf("%s: %d writes, expected %d", tc.name, writes, tc.writes)
		}
	}
}
