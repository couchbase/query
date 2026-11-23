//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package value

import (
	goerrors "errors"
	"testing"

	"github.com/couchbase/query/encryption"
	"github.com/couchbase/query/errors"
)

// A failure while spilling must be reported with its own error, not as a generic spill write error.
func TestAnnotatedArraySpillErrorKeepsCause(t *testing.T) {
	acquire := func(size int) AnnotatedValues { return make(AnnotatedValues, 0, size) }
	less := func(v1, v2 AnnotatedValue) bool { return v1.Collate(v2) < 0 }
	always := func(uint64, uint64) bool { return true }

	keyErr := errors.NewEncryptionError(errors.E_ENCRYPTION, goerrors.New("key unavailable"))
	aa := NewAnnotatedArray(acquire, nil, always, nil, less, false,
		func() (*encryption.EaRKey, errors.Error) { return nil, keyErr })
	defer aa.Release()

	if err := aa.Append(NewAnnotatedValue(NewValue(1))); err != nil {
		t.Fatalf("first append (nothing to spill) failed: %v", err)
	}
	err := aa.Append(NewAnnotatedValue(NewValue(2)))
	if err == nil {
		t.Fatalf("expected the encryption key error")
	}
	if err.Code() != errors.E_ENCRYPTION {
		t.Errorf("got code %v (%v), want %v", err.Code(), err, errors.E_ENCRYPTION)
	}

	// errors raised while spilling keep their own code
	noKey := func() (*encryption.EaRKey, errors.Error) { return nil, nil }

	// memory quota exceeded while releasing the spilled values
	quotaErr := errors.NewMemoryQuotaExceededError(2, 1)
	trackMem := func(size int64) error {
		if size < 0 {
			return quotaErr
		}
		return nil
	}
	aq := NewAnnotatedArray(acquire, nil, always, trackMem, less, false, noKey)
	defer aq.Release()
	if err := aq.Append(NewAnnotatedValue(NewValue(1))); err != nil {
		t.Fatalf("first append (nothing to spill) failed: %v", err)
	}
	err = aq.Append(NewAnnotatedValue(NewValue(2)))
	if err == nil || err.Code() != errors.E_MEMORY_QUOTA_EXCEEDED {
		t.Errorf("got %v, want code %v", err, errors.E_MEMORY_QUOTA_EXCEEDED)
	}

	// too many spill files: every append spills the previous value to a new file. All spill files stay open until
	// Truncate, so lower the limit rather than depend on the open file limit of the environment.
	defer func(n int) { _MAX_SPILL_FILES = n }(_MAX_SPILL_FILES)
	_MAX_SPILL_FILES = 5
	am := NewAnnotatedArray(acquire, nil, always, nil, less, false, noKey)
	defer am.Release()
	err = nil
	for i := 0; i <= _MAX_SPILL_FILES+2 && err == nil; i++ {
		err = am.Append(NewAnnotatedValue(NewValue(i)))
	}
	if err == nil || err.Code() != errors.E_VALUE_SPILL_MAX_FILES {
		t.Errorf("got %v, want code %v", err, errors.E_VALUE_SPILL_MAX_FILES)
	}

	// a plain error is still wrapped, and its text is reported
	e := spillError(errors.E_VALUE_SPILL_WRITE, goerrors.New("no space left on device"))
	if e.Code() != errors.E_VALUE_SPILL_WRITE || e.Error() != "Failed to write to spill file - cause: no space left on device" {
		t.Errorf("unexpected wrapped error: %v %q", e.Code(), e.Error())
	}
}
