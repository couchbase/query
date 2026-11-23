//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package errors

import (
	goerrors "errors"
	"strings"
	"testing"
)

func TestBucketActionErrorText(t *testing.T) {
	tests := []struct {
		name string
		e    any
		want string
	}{
		{"status", "DURABILITY_IMPOSSIBLE", "Unable to complete action after 1 attempts: DURABILITY_IMPOSSIBLE"},
		{"kv error map", map[string]any{"error": "SYNC_WRITE_AMBIGUOUS", "context": "ctx"},
			"Unable to complete action after 1 attempts: SYNC_WRITE_AMBIGUOUS (ctx)"},
		{"error", goerrors.New("connection reset"), "Unable to complete action after 1 attempts: connection reset"},
		{"nil", nil, "Unable to complete action after 1 attempts"},
	}
	for _, tt := range tests {
		err := NewBucketActionError(tt.e, 1)
		if got := err.Error(); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}

	// the status must also show up in a DML error built on top of it
	dml := NewCbDMLError(NewBucketActionError("DURABILITY_IMPOSSIBLE", 1), "Failed to perform UPSERT on key k",
		false, FALSE, "k", "ks")
	if !strings.Contains(dml.Error(), "DURABILITY_IMPOSSIBLE") {
		t.Errorf("DML error does not report the KV status: %q", dml.Error())
	}

	// a KV error map must keep its status for the KEY_ENOENT/KEY_EEXISTS checks on the cause
	ba := NewBucketActionError(map[string]any{"error": "KEY_ENOENT"}, 1)
	if !ba.ContainsText("KEY_ENOENT") {
		t.Errorf("bucket action error lost KEY_ENOENT")
	}
}

func TestValueErrorReportsCause(t *testing.T) {
	err := NewValueError(E_VALUE_SPILL_WRITE, goerrors.New("no space left on device"))
	if got, want := err.Error(), "Failed to write to spill file - cause: no space left on device"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
