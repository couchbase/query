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
	"slices"
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
	// Truncate, so lower the limit rather than depend on the open file limit of the environment. A sorted array merges its
	// spill files at the limit, so the error needs merging disabled.
	defer func(n, m int) { _MAX_SPILL_FILES, _SPILL_MERGE_FILES = n, m }(_MAX_SPILL_FILES, _SPILL_MERGE_FILES)
	_MAX_SPILL_FILES = 5
	_SPILL_MERGE_FILES = 0
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

// Spill files of a sorted array are merged as they accumulate and an unsorted array appends to a single spill file, so a
// large number of spills neither reaches the spill file limit nor changes the values or their order (sorted, or append
// order when unsorted), including when appending after reading the array.
func TestAnnotatedArraySpillMerge(t *testing.T) {
	defer func(n, m int) { _MAX_SPILL_FILES, _SPILL_MERGE_FILES = n, m }(_MAX_SPILL_FILES, _SPILL_MERGE_FILES)
	// every Append spills the previous value, so 2*n appends make about 2*n spills and about 2*n/2 merges of 3 files
	_MAX_SPILL_FILES = 6
	_SPILL_MERGE_FILES = 3

	acquire := func(size int) AnnotatedValues { return make(AnnotatedValues, 0, size) }
	always := func(uint64, uint64) bool { return true }
	less := func(v1, v2 AnnotatedValue) bool { return v1.Collate(v2) < 0 }

	const n = 500
	// also zlib compression without encryption (what MERGE uses), where an unsorted array keeps adding to the zlib stream
	noKey := func() (*encryption.EaRKey, errors.Error) { return nil, nil }
	modes := append(spillTestModes(t), spillTestMode{"compression enabled", true, noKey})
	for _, mode := range modes {
		for _, sorted := range []bool{true, false} {
			lessFn := less
			if !sorted {
				lessFn = nil
			}
			aa := NewAnnotatedArray(acquire, nil, always, nil, lessFn, mode.compress, mode.getEncryptionKey)

			var appended []int64

			// descending values, so that a sorted array has to reorder them
			appendRange := func(from, to int) {
				for i := from; i < to; i++ {
					v := i
					if sorted {
						v = 2*n - 1 - i
					}
					if err := aa.Append(NewAnnotatedValue(NewValue(v))); err != nil {
						t.Fatalf("%v sorted %v: append %d failed: %v", mode.name, sorted, i, err)
					}
					appended = append(appended, int64(v))
				}
				if len(aa.spill) > _MAX_SPILL_FILES || (!sorted && len(aa.spill) != 1) {
					t.Errorf("%v sorted %v: %v spill files", mode.name, sorted, len(aa.spill))
				}
			}

			check := func() {
				want := append([]int64(nil), appended...)
				if sorted {
					slices.Sort(want)
				}
				i := 0
				err := aa.Foreach(func(av AnnotatedValue) bool {
					if i >= len(want) || !av.Equals(NewValue(want[i])).Truth() {
						t.Errorf("%v sorted %v: value %d is %v (%T)", mode.name, sorted, i, av.Actual(), av.Actual())
						return false
					}
					i++
					return true
				})
				if err != nil || i != len(want) {
					t.Errorf("%v sorted %v: read %v values (error %v), want %v", mode.name, sorted, i, err, len(want))
				}
			}

			appendRange(0, n)
			check()
			// more appends after reading (Foreach reorders the spill files of a sorted array)
			appendRange(n, 2*n)
			check()
			aa.Release()
		}
	}
}

// Values sharing parents (e.g. UNNEST or JOIN results) keep their parent through spilling and merging.
func TestAnnotatedArraySpillMergeParents(t *testing.T) {
	defer func(n, m int) { _MAX_SPILL_FILES, _SPILL_MERGE_FILES = n, m }(_MAX_SPILL_FILES, _SPILL_MERGE_FILES)
	_MAX_SPILL_FILES = 6
	_SPILL_MERGE_FILES = 3

	acquire := func(size int) AnnotatedValues { return make(AnnotatedValues, 0, size) }
	always := func(uint64, uint64) bool { return true }
	less := func(v1, v2 AnnotatedValue) bool {
		k1, _ := v1.Field("k")
		k2, _ := v2.Field("k")
		return k1.Collate(k2) < 0
	}

	const n, nParents = 300, 7
	for _, mode := range spillTestModes(t) {
		parents := make([]Value, nParents)
		for i := range parents {
			parents[i] = NewValue(map[string]interface{}{"p": i})
		}
		aa := NewAnnotatedArray(acquire, nil, always, nil, less, mode.compress, mode.getEncryptionKey)
		for i := n - 1; i >= 0; i-- {
			sv := NewScopeValue(map[string]interface{}{"k": i}, parents[i%nParents])
			if err := aa.Append(NewAnnotatedValue(sv)); err != nil {
				t.Fatalf("%v: append %d failed: %v", mode.name, i, err)
			}
		}
		if len(aa.parentsMap) != nParents {
			t.Errorf("%v: %v parents kept, want %v", mode.name, len(aa.parentsMap), nParents)
		}
		i := 0
		err := aa.Foreach(func(av AnnotatedValue) bool {
			k, _ := av.Field("k")
			p, _ := av.Field("p")
			if !k.Equals(NewValue(i)).Truth() || !p.Equals(NewValue(i%nParents)).Truth() {
				t.Errorf("%v: value %d is k %v p %v", mode.name, i, k, p)
				return false
			}
			i++
			return true
		})
		if err != nil || i != n {
			t.Errorf("%v: read %v values (error %v), want %v", mode.name, i, err, n)
		}
		aa.Release()
	}
}

// Merging spill files does not change the memory accounting: the memory tracked while reading the array back is the
// same as without merges.
func TestAnnotatedArraySpillMergeTrackMemory(t *testing.T) {
	defer func(n, m int) { _MAX_SPILL_FILES, _SPILL_MERGE_FILES = n, m }(_MAX_SPILL_FILES, _SPILL_MERGE_FILES)
	_SPILL_MERGE_FILES = 3

	acquire := func(size int) AnnotatedValues { return make(AnnotatedValues, 0, size) }
	always := func(uint64, uint64) bool { return true }
	less := func(v1, v2 AnnotatedValue) bool { return v1.Collate(v2) < 0 }

	run := func(maxFiles int) int64 {
		_MAX_SPILL_FILES = maxFiles
		tracked := int64(0)
		trackMem := func(sz int64) error { tracked += sz; return nil }
		aa := NewAnnotatedArray(acquire, nil, always, trackMem, less, false, nil)
		aa.getEncryptionKey = func() (*encryption.EaRKey, errors.Error) { return nil, nil }
		for i := 0; i < 200; i++ {
			// as the callers do: the value is tracked when it is added
			av := NewAnnotatedValue(map[string]interface{}{"k": 200 - i, "s": "abc"})
			tracked += int64(av.Size())
			if err := aa.Append(av); err != nil {
				t.Fatalf("append %d failed: %v", i, err)
			}
		}
		err := aa.Foreach(func(av AnnotatedValue) bool { return true })
		if err != nil {
			t.Fatalf("max files %v: read failed: %v", maxFiles, err)
		}
		aa.Release()
		return tracked
	}
	noMerge := run(1000)
	merged := run(6)
	if merged != noMerge {
		t.Errorf("tracked memory with merges %v, without merges %v", merged, noMerge)
	}

	// the values held while merging count against the quota: the merge fails when the quota is exceeded
	_MAX_SPILL_FILES = 6
	quotaErr := errors.NewMemoryQuotaExceededError(2, 1)
	full := func(sz int64) error {
		if sz > 0 {
			return quotaErr
		}
		return nil
	}
	aa := NewAnnotatedArray(acquire, nil, always, full, less, false,
		func() (*encryption.EaRKey, errors.Error) { return nil, nil })
	defer aa.Release()
	var err errors.Error
	for i := 0; i < 20 && err == nil; i++ {
		err = aa.Append(NewAnnotatedValue(i))
	}
	if err == nil || err.Code() != errors.E_MEMORY_QUOTA_EXCEEDED {
		t.Errorf("got %v, want code %v", err, errors.E_MEMORY_QUOTA_EXCEEDED)
	}
	// a failed merge discards the array rather than leaving it without the merged files' values
	if len(aa.spill) != 0 || aa.Length() != 0 {
		t.Errorf("after the failed merge: %v spill files, length %v", len(aa.spill), aa.Length())
	}
}

// A sorted array read only partly (e.g. under a LIMIT) and then appended to, until its spill files are merged, still
// returns all values in order.
func TestAnnotatedArraySpillMergeAfterPartialRead(t *testing.T) {
	defer func(n, m int) { _MAX_SPILL_FILES, _SPILL_MERGE_FILES = n, m }(_MAX_SPILL_FILES, _SPILL_MERGE_FILES)
	_MAX_SPILL_FILES = 6
	_SPILL_MERGE_FILES = 3

	acquire := func(size int) AnnotatedValues { return make(AnnotatedValues, 0, size) }
	always := func(uint64, uint64) bool { return true }
	less := func(v1, v2 AnnotatedValue) bool { return v1.Collate(v2) < 0 }

	for _, mode := range spillTestModes(t) {
		aa := NewAnnotatedArray(acquire, nil, always, nil, less, mode.compress, mode.getEncryptionKey)
		const n = 100
		appendRange := func(from, to int) {
			for i := from; i < to; i++ {
				if err := aa.Append(NewAnnotatedValue(i)); err != nil {
					t.Fatalf("%v: append %d failed: %v", mode.name, i, err)
				}
			}
		}
		appendRange(n, 2*n)
		read := 0
		if err := aa.Foreach(func(AnnotatedValue) bool { read++; return read < 3 }); err != nil || read != 3 {
			t.Fatalf("%v: partial read: %v values, error %v", mode.name, read, err)
		}
		appendRange(0, n)
		i := 0
		err := aa.Foreach(func(av AnnotatedValue) bool {
			if !av.Equals(NewValue(i)).Truth() {
				t.Errorf("%v: value %d is %v", mode.name, i, av.Actual())
				return false
			}
			i++
			return true
		})
		if err != nil || i != 2*n {
			t.Errorf("%v: read %v values (error %v), want %v", mode.name, i, err, 2*n)
		}
		aa.Release()
	}
}
