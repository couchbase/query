//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package prepareds

import (
	"fmt"
	"testing"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/settings"
)

// MB-75506: with plan stability off, a plan without SAVE distributed by another node, replacing a cached plan
// saved with PREPARE SAVE, removes the saved plan, if it is still the cached plan
func TestAddDistributedPrepared(t *testing.T) {
	defer func(d func(string) errors.Error, v func(*plan.Prepared) errors.Error,
		i func(string) (savedPlan, errors.Error),
		l func(string, settings.PlanStabilityMode, settings.PlanStabilityErrorPolicy) (*plan.Prepared,
			errors.Error)) {
		deletePreparedFn = d
		verifyPreparedFn = v
		savedPlanInfoFn = i
		loadPreparedFn = l
		clearAllSaved()
	}(deletePreparedFn, verifyPreparedFn, savedPlanInfoFn, loadPreparedFn)

	var deleteErr errors.Error
	deletes := 0
	deletePreparedFn = func(string) errors.Error {
		deletes++
		return deleteErr
	}
	verifyPreparedFn = func(*plan.Prepared) errors.Error {
		return nil
	}
	// the plan comes with the request: never looked for in QUERY_METADATA
	loads := 0
	loadPreparedFn = func(string, settings.PlanStabilityMode, settings.PlanStabilityErrorPolicy) (*plan.Prepared,
		errors.Error) {
		loads++
		return nil, nil
	}

	const name = "dp1"
	const text = "PREPARE dp1 FROM SELECT 1"
	newPrepared := func(text string, save bool, version int) *plan.Prepared {
		prepared := plan.NewPrepared(plan.NewSequence(plan.NewDummyScan(0, 0, 0, 0)), nil, nil, nil, nil)
		prepared.SetName(name)
		prepared.SetText(text)
		prepared.SetPersist(save)
		// a different plan for the same statement
		prepared.SetIndexApiVersion(version)
		return prepared
	}

	// the saved plan, as found in QUERY_METADATA
	const (
		_SAVED_CACHED  = iota // the cached plan
		_SAVED_NEWER          // a newer plan saved with PREPARE SAVE
		_SAVED_NO_SAVE        // a plan without SAVE
		_SAVED_MISSING        // none
		_SAVED_READ_ERROR
	)

	for _, tc := range []struct {
		name        string
		mode        settings.PlanStabilityMode
		cachedText  string // no cached plan if empty
		cachedSave  bool
		save        bool // the distributed plan has SAVE
		samePlan    bool // the distributed plan is the cached plan
		saved       int
		deleteFails bool
		deletes     int
		err         bool
		replaced    bool
		knownSaved  bool
	}{
		{"saved, replaced without SAVE", settings.PS_MODE_OFF, text, true, false, false, _SAVED_CACHED, false,
			1, false, true, false},
		{"saved, removing it fails", settings.PS_MODE_OFF, text, true, false, false, _SAVED_CACHED, true,
			1, false, true, true},
		{"saved, a newer plan saved", settings.PS_MODE_OFF, text, true, false, false, _SAVED_NEWER, false,
			0, false, true, true},
		{"saved, now saved without SAVE", settings.PS_MODE_OFF, text, true, false, false, _SAVED_NO_SAVE, false,
			0, false, true, true},
		{"saved, no longer saved", settings.PS_MODE_OFF, text, true, false, false, _SAVED_MISSING, false,
			0, false, true, false},
		{"saved, reading it fails", settings.PS_MODE_OFF, text, true, false, false, _SAVED_READ_ERROR, false,
			0, false, true, true},
		{"saved, replaced with SAVE", settings.PS_MODE_OFF, text, true, true, false, _SAVED_CACHED, false,
			0, false, true, true},
		{"not saved", settings.PS_MODE_OFF, text, false, false, false, _SAVED_MISSING, false,
			0, false, true, false},
		{"not cached", settings.PS_MODE_OFF, "", false, false, false, _SAVED_MISSING, false,
			0, false, true, false},
		{"same plan", settings.PS_MODE_OFF, text, true, true, true, _SAVED_CACHED, false,
			0, false, false, true},
		{"saved, different statement", settings.PS_MODE_OFF, "PREPARE dp1 FROM SELECT 2", true, false, false,
			_SAVED_CACHED, false, 0, true, false, true},
		{"saved, plan stability on", settings.PS_MODE_PREPARED_ONLY, text, true, false, false, _SAVED_CACHED,
			false, 0, false, true, true},
		{"not cached, plan stability on", settings.PS_MODE_PREPARED_ONLY, "", false, false, false, _SAVED_MISSING,
			false, 0, false, true, false},
	} {
		// replaces the package-global prepared cache: fine for tests in this package, which run in sequence
		PreparedsInit(16)
		clearAllSaved()
		deletes = 0
		loads = 0
		deleteErr = nil
		if tc.deleteFails {
			deleteErr = errors.NewPreparedDeletePlanError(name, fmt.Errorf("QUERY_METADATA not available"))
		}
		var cached *plan.Prepared
		if tc.cachedText != "" {
			cached = newPrepared(tc.cachedText, tc.cachedSave, 1)
			if _, er := cached.BuildEncodedPlan(); er != nil {
				t.Fatalf("%s: encoding: %v", tc.name, er)
			}
			prepareds.add(cached, true, false, nil)
		}

		savedKind := tc.saved
		savedPlanInfoFn = func(string) (savedPlan, errors.Error) {
			switch savedKind {
			case _SAVED_CACHED:
				return savedPlan{text: text, encoded: cached.EncodedPlan(), persist: true, found: true,
					readable: true}, nil
			case _SAVED_NEWER:
				return savedPlan{text: text, encoded: "newer", persist: true, found: true, readable: true}, nil
			case _SAVED_NO_SAVE:
				return savedPlan{text: text, encoded: cached.EncodedPlan(), found: true, readable: true}, nil
			case _SAVED_READ_ERROR:
				return savedPlan{}, errors.NewQueryMetadataError("test", nil)
			}
			return savedPlan{}, nil
		}

		distributed := newPrepared(text, tc.save, 2)
		if tc.samePlan {
			distributed = cached
		}
		encoded, er := distributed.BuildEncodedPlan()
		if er != nil {
			t.Fatalf("%s: encoding: %v", tc.name, er)
		}

		err := AddDistributedPrepared(name, encoded, tc.mode, settings.PS_ERROR_MODERATE, logging.NULL_LOG)
		ce := prepareds.get(name, false)
		replaced := ce != nil && ce.Prepared != cached && ce.Prepared.EncodedPlan() == encoded
		if deletes != tc.deletes || (err != nil) != tc.err || replaced != tc.replaced || loads != 0 {
			t.Errorf("%s: %d deletes, error %v, replaced %v, %d loads; expected %d deletes, error %v, replaced %v, "+
				"no loads", tc.name, deletes, err, replaced, loads, tc.deletes, tc.err, tc.replaced)
		}
		if saved := isSaved(name); saved != tc.knownSaved {
			t.Errorf("%s: known as saved %v, expected %v", tc.name, saved, tc.knownSaved)
		}
	}
}
