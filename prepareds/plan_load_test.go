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

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/planner"
	"github.com/couchbase/query/settings"
)

func TestCheckSavedPlan(t *testing.T) {
	// start with no names tracked, whatever the tests run before left (the order of the tests is not fixed)
	defer restorePlanLoadState(savePlanLoadState())
	restorePlanLoadState(planLoadSnapshot{unloaded: map[string]bool{}, standIns: map[string]*standIn{},
		saved: map[string]bool{}})

	PreparedsInit(2)
	defer endPriming()

	const name = "p1(default:b.s)"
	modes := []settings.PlanStabilityMode{settings.PS_MODE_OFF, settings.PS_MODE_PREPARED_ONLY,
		settings.PS_MODE_AD_HOC, settings.PS_MODE_AD_HOC_READ_ONLY}

	// MB-75506: not priming, nothing unloaded: look for a saved plan only if one can exist for the statement,
	// whatever the cache state
	for _, mode := range modes {
		adHoc := mode == settings.PS_MODE_AD_HOC || mode == settings.PS_MODE_AD_HOC_READ_ONLY
		for _, tc := range []struct {
			kind     lookupKind
			expected bool
		}{
			{_LOOKUP_PREPARED, mode != settings.PS_MODE_OFF},
			{_LOOKUP_AD_HOC, adHoc},
			{_LOOKUP_CACHE_ONLY, false},
		} {
			if checkSavedPlan(name, tc.kind, mode) != tc.expected {
				t.Errorf("mode %v, kind %v: expected check for saved plan %v", mode, tc.kind, tc.expected)
			}
		}
	}

	// with plan stability off, a prepared statement known on this node to be saved with PREPARE SAVE
	markSaved(name)
	if !checkSavedPlan(name, _LOOKUP_PREPARED, settings.PS_MODE_OFF) {
		t.Fatalf("expected check for a plan saved with PREPARE SAVE, plan stability off")
	}
	if checkSavedPlan(name, _LOOKUP_AD_HOC, settings.PS_MODE_OFF) {
		t.Fatalf("expected no check for an ad hoc statement, plan stability off")
	}
	clearSaved(name)
	if isSaved(name) || planLoad.savedSize != 0 {
		t.Fatalf("expected name no longer saved")
	}

	// while priming, every miss looks for a saved plan
	PreparedsStartPriming()
	if !checkSavedPlan(name, _LOOKUP_CACHE_ONLY, settings.PS_MODE_OFF) ||
		!checkSavedPlan("other", _LOOKUP_AD_HOC, settings.PS_MODE_OFF) {
		t.Fatalf("expected check for saved plan while priming")
	}
	endPriming()

	// a saved plan that could not be put in the cache
	markUnloaded(name)
	markUnloaded(name)
	if !checkSavedPlan(name, _LOOKUP_CACHE_ONLY, settings.PS_MODE_OFF) {
		t.Fatalf("expected check for unloaded saved plan")
	}
	if checkSavedPlan("other", _LOOKUP_CACHE_ONLY, settings.PS_MODE_OFF) {
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

	// MB-75506: adding a plan saved with PREPARE SAVE marks it saved, which stays after it is evicted
	saved := plan.NewPrepared(nil, nil, nil, nil, nil)
	saved.SetName("ps")
	saved.SetPersist(true)
	prepareds.add(saved, false, false, nil)
	PreparedsSetLimit(1)
	other := plan.NewPrepared(nil, nil, nil, nil, nil)
	other.SetName("pother")
	prepareds.add(other, false, false, nil)
	PreparedsSetLimit(2)
	if prepareds.get("ps", false) != nil || !isSaved("ps") || isSaved("pother") {
		t.Fatalf("expected the plan saved with PREPARE SAVE evicted, and known as saved")
	}
	clearAllSaved()
	if isSaved("ps") || planLoad.savedSize != 0 {
		t.Fatalf("expected no name known as saved")
	}

	// explicit clear (nothing saved under the name)
	markUnloaded("p2")
	clearUnloaded("p2")
	if isUnloaded("p2") {
		t.Fatalf("expected name cleared")
	}

	// MB-75506: a full cache no longer makes a difference
	prepared2 := plan.NewPrepared(nil, nil, nil, nil, nil)
	prepared2.SetName("p3")
	prepareds.add(prepared2, false, false, nil)
	if checkSavedPlan("other", _LOOKUP_AD_HOC, settings.PS_MODE_OFF) {
		t.Fatalf("expected no check for an ad hoc statement with plan stability off, cache full")
	}
}

// MB-75506: a cache miss looks for a saved plan whenever one can exist for the statement, cache full or not
func TestGetPreparedLookup(t *testing.T) {
	defer func(l func(string, settings.PlanStabilityMode, settings.PlanStabilityErrorPolicy) (*plan.Prepared,
		errors.Error)) {
		loadPreparedFn = l
	}(loadPreparedFn)

	// replaces the package-global prepared cache: fine for tests in this package, which run in sequence
	PreparedsInit(16)

	loads := 0
	var saved *plan.Prepared
	loadPreparedFn = func(string, settings.PlanStabilityMode, settings.PlanStabilityErrorPolicy) (*plan.Prepared,
		errors.Error) {
		loads++
		return saved, nil
	}

	modes := []settings.PlanStabilityMode{settings.PS_MODE_OFF, settings.PS_MODE_PREPARED_ONLY,
		settings.PS_MODE_AD_HOC, settings.PS_MODE_AD_HOC_READ_ONLY}
	for _, mode := range modes {
		adHoc := mode == settings.PS_MODE_AD_HOC || mode == settings.PS_MODE_AD_HOC_READ_ONLY
		for _, tc := range []struct {
			kind    string
			options uint32
			loads   int
		}{
			{"prepared", OPT_TRACK | OPT_VERIFY, map[bool]int{true: 1, false: 0}[mode != settings.PS_MODE_OFF]},
			{"ad hoc", OPT_TRACK | OPT_VERIFY | OPT_ADHOC, map[bool]int{true: 1, false: 0}[adHoc]},
			{"cache only", OPT_CACHE_ONLY, 0},
		} {
			// nothing saved: no such prepared
			loads = 0
			saved = nil
			prepared, err := getPrepared("missing", "", nil, tc.options, nil, mode, settings.PS_ERROR_MODERATE,
				nil, datastore.UNBOUNDED)
			if prepared != nil || err == nil || err.Code() != errors.E_NO_SUCH_PREPARED || loads != tc.loads {
				t.Errorf("mode %v, %s, nothing saved: got %v, %v, %d loads; expected no such prepared, %d loads",
					mode, tc.kind, prepared, err, loads, tc.loads)
			}

			// a saved plan not in the cache is returned
			loads = 0
			saved = plan.NewPrepared(nil, nil, nil, nil, nil)
			prepared, err = getPrepared("missing", "", nil, tc.options, nil, mode, settings.PS_ERROR_MODERATE,
				nil, datastore.UNBOUNDED)
			if tc.loads > 0 && (prepared != saved || err != nil) {
				t.Errorf("mode %v, %s: got %v, %v; expected the saved plan", mode, tc.kind, prepared, err)
			}
		}

		// with plan stability off, a prepared statement known on this node to be saved with PREPARE SAVE is
		// looked for; when nothing is found, it is no longer known as saved
		markSaved("psaved")
		loads = 0
		saved = nil
		prepared, err := getPrepared("psaved", "", nil, OPT_TRACK|OPT_VERIFY, nil, mode, settings.PS_ERROR_MODERATE, nil,
			datastore.UNBOUNDED)
		if loads != 1 || err == nil || err.Code() != errors.E_NO_SUCH_PREPARED || isSaved("psaved") {
			t.Errorf("mode %v, saved with PREPARE SAVE: %d loads, %v, saved %v; expected 1 load, no such prepared, "+
				"not saved", mode, loads, err, isSaved("psaved"))
		}

		// an ad hoc statement (GetAutoPreparePlan()) is looked for under AD_HOC and AD_HOC_READ_ONLY
		var prepContext planner.PrepareContext
		planner.NewPrepareContext(&prepContext, "req1", "", nil, nil, 0, 0, false, false, nil, nil, nil, true, mode,
			settings.PS_ERROR_MODERATE, datastore.UNBOUNDED)
		loads = 0
		saved = nil
		prepared, err = GetAutoPreparePlan("adhoc1", "SELECT 1", "default", &prepContext)
		expected := map[bool]int{true: 1, false: 0}[adHoc]
		if prepared != nil || err != nil || loads != expected {
			t.Errorf("mode %v, auto prepare: got %v, %v, %d loads; expected none, %d loads", mode, prepared, err,
				loads, expected)
		}

		// an inline UDF entry is looked for once, with plan stability on
		loads = 0
		saved = nil
		prepared, err = GetUdfPrepared("udf1", false, mode, settings.PS_ERROR_MODERATE, nil)
		expected = map[bool]int{true: 1, false: 0}[mode != settings.PS_MODE_OFF]
		if prepared != nil || err != nil || loads != expected {
			t.Errorf("mode %v, inline UDF: got %v, %v, %d loads; expected none, %d loads", mode, prepared, err,
				loads, expected)
		}
	}
}

// MB-75515: inline UDF entries are not prepared statements: their names are reserved, and they can not be executed
func TestInlineUdfEntryNames(t *testing.T) {
	defer func(l func(string, settings.PlanStabilityMode, settings.PlanStabilityErrorPolicy) (*plan.Prepared,
		errors.Error)) {
		loadPreparedFn = l
	}(loadPreparedFn)
	loadPreparedFn = func(string, settings.PlanStabilityMode, settings.PlanStabilityErrorPolicy) (*plan.Prepared,
		errors.Error) {
		return nil, nil
	}
	PreparedsInit(16)

	name := GetUdfPreparedName("default:f1")
	if !prepareds.IsPredefinedPrepareName(name) || prepareds.IsPredefinedPrepareName("p1") {
		t.Errorf("expected the names of inline UDF entries, and only them, reserved")
	}

	entry := plan.NewPrepared(nil, nil, nil, nil, nil)
	entry.SetName(name)
	entry.SetInlineUdf()
	prepareds.add(entry, true, false, nil)

	// EXECUTE (and auto_execute, the prepared request parameter): no such prepared statement
	prepared, err := GetPreparedWithContext(name, "", nil, OPT_TRACK, nil, settings.PS_MODE_PREPARED_ONLY,
		settings.PS_ERROR_MODERATE, datastore.UNBOUNDED)
	if prepared != nil || err == nil || err.Code() != errors.E_NO_SUCH_PREPARED {
		t.Errorf("EXECUTE of an inline UDF entry: got %v, %v; expected no such prepared statement", prepared, err)
	}

	// the inline UDF still gets its entry
	if prepared, err = GetUdfPrepared("default:f1", false, settings.PS_MODE_PREPARED_ONLY, settings.PS_ERROR_MODERATE,
		nil); prepared != entry || err != nil {
		t.Errorf("inline UDF: got %v, %v; expected its entry", prepared, err)
	}
}
