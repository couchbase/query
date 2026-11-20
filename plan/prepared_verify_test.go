//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import (
	"sync"
	"testing"
)

// MB-75478: the verification error indicators are not saved with the plan (encoded_plan); SAVE is
func TestVerificationErrorNotEncoded(t *testing.T) {
	prepared := NewPrepared(NewSequence(NewDummyScan(0, 0, 0, 0)), nil, nil, nil, nil)
	prepared.SetName("p1")
	prepared.SetText("SELECT 1")
	prepared.SetPersist(true)
	prepared.SetFatalError(true)
	prepared.SetErrorCount(3)

	encoded, err := prepared.BuildEncodedPlan()
	if err != nil {
		t.Fatalf("encoding: %v", err)
	}
	decoded, _, er := NewPreparedFromEncodedPlan(encoded, false)
	if er != nil {
		t.Fatalf("decoding: %v", er)
	}
	if decoded.HasVerificationError() || decoded.HasFatalError() || decoded.ErrorCount() != 0 {
		t.Errorf("decoded plan has a verification error: fatal %v, count %d", decoded.HasFatalError(),
			decoded.ErrorCount())
	}
	if !decoded.Persist() {
		t.Errorf("decoded plan lost SAVE")
	}
	if !prepared.HasFatalError() || prepared.ErrorCount() != 3 {
		t.Errorf("the plan lost its verification error: fatal %v, count %d", prepared.HasFatalError(),
			prepared.ErrorCount())
	}
}

// MB-75478: concurrent executions update and read the verification error indicators, and SAVE, of a cached
// plan (run with -race)
func TestPreparedConcurrentAccess(t *testing.T) {
	prepared := NewPrepared(nil, nil, nil, nil, nil)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				prepared.errCount.Add(1)
				_ = prepared.HasVerificationError()
				_ = prepared.ErrorCount()
				prepared.SetPersist(j%2 == 0)
				_ = prepared.Persist()
				if i == 0 && j == 500 {
					prepared.SetFatalError(true)
				}
			}
		}(i)
	}
	wg.Wait()
	if prepared.ErrorCount() != 8000 || !prepared.HasFatalError() {
		t.Errorf("error count %d, fatal %v", prepared.ErrorCount(), prepared.HasFatalError())
	}
}
