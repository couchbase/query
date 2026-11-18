//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package prepareds

import (
	"testing"

	"github.com/couchbase/query/plan"
)

func TestCheckSavedPlan(t *testing.T) {
	PreparedsInit(2)
	defer endPriming()

	const name = "p1(default:b.s)"

	// not priming, nothing unloaded, cache not full: no need to look for a saved plan
	if checkSavedPlan(name) {
		t.Fatalf("expected no check for saved plan")
	}

	// while priming, every miss looks for a saved plan
	PreparedsStartPriming()
	if !checkSavedPlan(name) || !checkSavedPlan("other") {
		t.Fatalf("expected check for saved plan while priming")
	}
	endPriming()

	// a saved plan that could not be put in the cache
	markUnloaded(name)
	markUnloaded(name)
	if !checkSavedPlan(name) {
		t.Fatalf("expected check for unloaded saved plan")
	}
	if checkSavedPlan("other") {
		t.Fatalf("expected no check for a name not unloaded")
	}

	// adding the statement to the cache clears it
	prepared := plan.NewPrepared(nil, nil, nil, nil, nil)
	prepared.SetName("p1")
	prepared.SetQueryContext("default:b.s")
	prepareds.add(prepared, false, false, nil)
	if isUnloaded(name) {
		t.Fatalf("expected name cleared once added to the cache")
	}
	if planLoad.unloadedSize != 0 {
		t.Fatalf("expected unloaded size 0, got %d", planLoad.unloadedSize)
	}

	// explicit clear (nothing saved under the name)
	markUnloaded("p2")
	clearUnloaded("p2")
	if isUnloaded("p2") {
		t.Fatalf("expected name cleared")
	}

	// a full cache (entries may have been evicted) always looks for a saved plan
	prepared2 := plan.NewPrepared(nil, nil, nil, nil, nil)
	prepared2.SetName("p3")
	prepareds.add(prepared2, false, false, nil)
	if !checkSavedPlan("other") {
		t.Fatalf("expected check for saved plan when the cache is full")
	}
}
