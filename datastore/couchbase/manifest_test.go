//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package couchbase

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestManifestsBehind(t *testing.T) {
	tests := []struct {
		name             string
		maniUid, newMani uint64
		haveExternal     bool
		extUid, newExt   uint64
		expected         bool
	}{
		{"up to date", 51, 51, true, 42, 42, false},
		{"collections manifest behind", 50, 51, true, 42, 42, true},
		{"external manifest behind", 51, 51, true, 41, 42, true},
		{"newer than announced", 52, 51, true, 43, 42, false},

		// external collections not available: the bucket still announces an external uid, but there is
		// no external manifest to read, so waiting for it would never end
		{"no external manifest", 51, 51, false, 0, 42, false},
		{"no external manifest, collections behind", 50, 51, false, 0, 42, true},
	}
	for _, tt := range tests {
		if got := manifestsBehind(tt.maniUid, tt.newMani, tt.haveExternal, tt.extUid, tt.newExt); got != tt.expected {
			t.Errorf("%s: manifestsBehind = %v, expected %v", tt.name, got, tt.expected)
		}
	}
}

func TestExternalManifestUnreadable(t *testing.T) {
	tests := []struct {
		name         string
		haveExternal bool
		newExt       uint64
		expected     bool
	}{
		{"external manifest read", true, 42, false},
		{"announced but not read", false, 42, true},
		{"none announced", false, 0, false},
	}
	for _, tt := range tests {
		if got := externalManifestUnreadable(tt.haveExternal, tt.newExt); got != tt.expected {
			t.Errorf("%s: externalManifestUnreadable = %v, expected %v", tt.name, got, tt.expected)
		}
	}
}

func TestManifestRetryDelay(t *testing.T) {
	var total time.Duration
	prev := time.Duration(0)
	for retries := 0; retries < _MANIFEST_READ_RETRIES; retries++ {
		d := manifestRetryDelay(retries)
		if d < prev || d > _MANIFEST_RETRY_MAX_DELAY {
			t.Errorf("retry %v: delay %v, previous %v, cap %v", retries, d, prev, _MANIFEST_RETRY_MAX_DELAY)
		}
		prev = d
		total += d
	}

	// long enough to allow for a manifest lagging the announced uid, short enough not to hold a request
	if total < 500*time.Millisecond || total > time.Second {
		t.Errorf("total wait over %v re-reads is %v, expected between 0.5s and 1s", _MANIFEST_READ_RETRIES, total)
	}
	if d := manifestRetryDelay(1000); d != _MANIFEST_RETRY_MAX_DELAY {
		t.Errorf("delay after many reads is %v, expected %v", d, _MANIFEST_RETRY_MAX_DELAY)
	}
}

func TestLogDue(t *testing.T) {
	var last atomic.Int64
	now := time.Now().UnixNano()
	interval := time.Minute

	if !logDue(&last, now, interval) {
		t.Errorf("first message not logged")
	}
	if logDue(&last, now+int64(time.Second), interval) {
		t.Errorf("message logged again within the interval")
	}
	if !logDue(&last, now+int64(interval), interval) {
		t.Errorf("message not logged after the interval")
	}

	// a successful refresh resets the time, so that the next problem is logged straight away
	last.Store(0)
	if !logDue(&last, now+int64(interval)+int64(time.Second), interval) {
		t.Errorf("message not logged after a reset")
	}
}
