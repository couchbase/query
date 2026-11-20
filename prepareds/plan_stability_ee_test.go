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

// MB-75506: turning plan stability on waits until QUERY_METADATA serves reads, and fails if it does not in time
func TestWaitQueryMetadataReads(t *testing.T) {
	defer func(h func(bool, string, string, bool) (bool, errors.Error), r func() errors.Error, i, m time.Duration) {
		hasQueryMetadataFn = h
		readQueryMetadataFn = r
		readRetryInterval = i
		readMaxWait = m
	}(hasQueryMetadataFn, readQueryMetadataFn, readRetryInterval, readMaxWait)

	hasQueryMetadataFn = func(bool, string, string, bool) (bool, errors.Error) {
		return true, nil
	}

	readRetryInterval = time.Millisecond
	readMaxWait = 200 * time.Millisecond

	for _, tc := range []struct {
		name     string
		failures int // reads that fail before reads succeed (-1: never)
		err      bool
	}{
		{"ready", 0, false},
		{"ready after retries", 3, false},
		{"never ready", -1, true},
	} {
		reads := 0
		failures := tc.failures
		readQueryMetadataFn = func() errors.Error {
			reads++
			if failures < 0 || reads <= failures {
				return errors.NewQueryMetadataError("Error performing bulk get operation", nil)
			}
			return nil
		}
		// turning plan stability on, QUERY_METADATA exists
		err := prepareds.EnsureQueryMetadata("req1")
		if (err != nil) != tc.err {
			t.Errorf("%s: error %v, expected error %v", tc.name, err, tc.err)
		}
		if tc.failures >= 0 && reads != tc.failures+1 {
			t.Errorf("%s: %d reads, expected %d", tc.name, reads, tc.failures+1)
		}
	}
}
