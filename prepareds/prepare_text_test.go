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

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/parser/n1ql"
	"github.com/couchbase/query/plan"
)

// the PREPARE options are removed from the text whatever the case and the whitespace between them
func TestGetText(t *testing.T) {
	for _, tc := range []struct{ stmt, text string }{
		{"PREPARE p1 FROM SELECT 1", "PREPARE p1 FROM SELECT 1"},
		{"PREPARE SAVE p1 FROM SELECT 1", "PREPARE p1 FROM SELECT 1"},
		{"PREPARE FORCE p1 FROM SELECT 1", "PREPARE p1 FROM SELECT 1"},
		{"PREPARE FORCE SAVE p1 FROM SELECT 1", "PREPARE p1 FROM SELECT 1"},
		{"PREPARE SAVE FORCE p1 FROM SELECT 1", "PREPARE p1 FROM SELECT 1"},
		{"PREPARE FORCE  SAVE p1 FROM SELECT 1", "PREPARE p1 FROM SELECT 1"},
		{"PREPARE SAVE\t\tFORCE p1 FROM SELECT 1", "PREPARE p1 FROM SELECT 1"},
		{"PREPARE\tSAVE p1 FROM SELECT 1", "PREPARE p1 FROM SELECT 1"},
		{"PREPARE\nFORCE\nSAVE p1 FROM SELECT 1", "PREPARE p1 FROM SELECT 1"},
		{"prepare force save p1 as select 1", "prepare p1 as select 1"},
		{"PREPARE  SAVE p1 FROM SELECT 1", "PREPARE p1 FROM SELECT 1"},
		{"PREPARE FORCE\t ps8 FROM SELECT 1", "PREPARE\t ps8 FROM SELECT 1"}, // after the options: kept
		{"PREPARE SAVE SELECT 1", "PREPARE SELECT 1"},
		{"PREPARE savepoint1 FROM SELECT 1", "PREPARE savepoint1 FROM SELECT 1"},
		{"PREPARE SAVE forced FROM SELECT 1", "PREPARE forced FROM SELECT 1"},
	} {
		stmt, err := n1ql.ParseStatement2(tc.stmt, "default", "")
		if err != nil {
			t.Errorf("%q: parse error %v", tc.stmt, err)
			continue
		}
		prepare := stmt.(*algebra.Prepare)
		text := prepareds.GetText(prepare.Text(), prepare.Offset())
		if text != tc.text {
			t.Errorf("%q: text %q, expected %q", tc.stmt, text, tc.text)
		}
		if _, err = n1ql.ParseStatement2(text, "default", ""); err != nil {
			t.Errorf("%q: text %q does not parse: %v", tc.stmt, text, err)
		}
	}
}

// a new plan for a PREPARE without SAVE, plan stability off, replacing a plan saved with PREPARE SAVE: the saved
// plan is removed
func TestAddPreparedStatement(t *testing.T) {
	defer func(d func(string) errors.Error, p func(*plan.Prepared) errors.Error,
		i func(string) (savedPlan, errors.Error)) {
		deletePreparedFn = d
		persistPreparedFn = p
		savedPlanInfoFn = i
	}(deletePreparedFn, persistPreparedFn, savedPlanInfoFn)

	var deleteErr errors.Error
	deletes := 0
	deletePreparedFn = func(string) errors.Error {
		deletes++
		return deleteErr
	}
	persistPreparedFn = func(*plan.Prepared) errors.Error {
		return nil
	}
	// MB-75506: a name not cached is looked for in QUERY_METADATA: nothing saved
	savedPlanInfoFn = func(string) (savedPlan, errors.Error) {
		return savedPlan{}, nil
	}

	const text = "PREPARE ps1 FROM SELECT 1"
	newPrepared := func(text string, save bool) *plan.Prepared {
		prepared := plan.NewPrepared(nil, nil, nil, nil, nil)
		prepared.SetName("ps1")
		prepared.SetText(text)
		prepared.SetPersist(save)
		return prepared
	}

	for _, tc := range []struct {
		name          string
		cachedText    string // no cached plan if empty
		cachedSave    bool
		save          bool // the new plan has SAVE
		planStability bool
		deleteFails   bool
		deletes       int
		err           bool
		replaced      bool
	}{
		{"saved, replaced without SAVE", text, true, false, false, false, 1, false, true},
		{"saved, delete fails", text, true, false, false, true, 1, true, false},
		{"saved, replaced with SAVE", text, true, true, false, false, 0, false, true},
		{"saved, plan stability on", text, true, false, true, false, 0, false, true},
		{"cached, not saved", text, false, false, false, false, 0, false, true},
		{"not cached", "", false, false, false, false, 0, false, true},
		{"saved, different statement", "PREPARE ps1 FROM SELECT 2", true, false, false, false, 0, true, false},
	} {
		// replaces the package-global prepared cache: fine for tests in this package, which run in sequence
		PreparedsInit(16)
		deletes = 0
		deleteErr = nil
		if tc.deleteFails {
			deleteErr = errors.NewPreparedDeletePlanError("ps1", fmt.Errorf("QUERY_METADATA not available"))
		}
		if tc.cachedText != "" {
			prepareds.add(newPrepared(tc.cachedText, tc.cachedSave), false, false, nil)
		}

		prepared := newPrepared(text, tc.save)
		err := AddPreparedStatement(prepared, tc.planStability)
		ce := prepareds.get("ps1", false)
		replaced := ce != nil && ce.Prepared == prepared
		if deletes != tc.deletes || (err != nil) != tc.err || replaced != tc.replaced {
			t.Errorf("%s: %d deletes, error %v, replaced %v; expected %d deletes, error %v, replaced %v", tc.name,
				deletes, err, replaced, tc.deletes, tc.err, tc.replaced)
		}
	}
}

// the name is not cached (e.g. evicted): the saved plan is checked the same way as a cached one
func TestAddPreparedStatementNotCached(t *testing.T) {
	defer func(d func(string) errors.Error, i func(string) (savedPlan, errors.Error)) {
		deletePreparedFn = d
		savedPlanInfoFn = i
	}(deletePreparedFn, savedPlanInfoFn)

	deletes := 0
	deletePreparedFn = func(string) errors.Error {
		deletes++
		return nil
	}

	const text = "PREPARE ps2 FROM SELECT 1"
	for _, tc := range []struct {
		name       string
		savedText  string // no saved plan if empty
		savedSave  bool
		unreadable bool
		infoErr    bool
		deletes    int
		err        bool
		replaced   bool
	}{
		{"saved with SAVE", text, true, false, false, 1, false, true},
		{"saved without SAVE", text, false, false, false, 0, false, true},
		{"not saved", "", false, false, false, 0, false, true},
		{"saved, different statement", "PREPARE ps2 FROM SELECT 2", true, false, false, 0, true, false},
		{"saved, can not be read", "", false, true, false, 1, false, true},
		{"reading the saved plan fails", "", false, false, true, 0, true, false},
	} {
		// replaces the package-global prepared cache: fine for tests in this package, which run in sequence
		PreparedsInit(16)
		deletes = 0
		savedText, savedSave, unreadable, infoErr := tc.savedText, tc.savedSave, tc.unreadable, tc.infoErr
		savedPlanInfoFn = func(string) (savedPlan, errors.Error) {
			if infoErr {
				return savedPlan{}, errors.NewQueryMetadataError("test", nil)
			} else if unreadable {
				return savedPlan{found: true}, nil
			}
			return savedPlan{text: savedText, persist: savedSave, found: savedText != "", readable: true}, nil
		}

		// MB-75506: with plan stability off, a name known on this node to be saved with PREPARE SAVE (e.g.
		// evicted) is looked for, whatever the cache state
		markSaved("ps2")
		prepared := plan.NewPrepared(nil, nil, nil, nil, nil)
		prepared.SetName("ps2")
		prepared.SetText(text)
		err := AddPreparedStatement(prepared, false)
		ce := prepareds.get("ps2", false)
		replaced := ce != nil && ce.Prepared == prepared
		if deletes != tc.deletes || (err != nil) != tc.err || replaced != tc.replaced {
			t.Errorf("%s: %d deletes, error %v, replaced %v; expected %d deletes, error %v, replaced %v", tc.name,
				deletes, err, replaced, tc.deletes, tc.err, tc.replaced)
		}
		clearAllSaved()
	}

	// MB-75506: with plan stability off, a name not known on this node to be saved with PREPARE SAVE is not
	// looked for (e.g. saved on another node, whose distribution did not reach this node)
	PreparedsInit(16)
	deletes = 0
	reads := 0
	savedPlanInfoFn = func(string) (savedPlan, errors.Error) {
		reads++
		return savedPlan{text: text, persist: true, found: true, readable: true}, nil
	}
	prepared := plan.NewPrepared(nil, nil, nil, nil, nil)
	prepared.SetName("ps2")
	prepared.SetText(text)
	if err := AddPreparedStatement(prepared, false); err != nil || reads != 0 || deletes != 0 {
		t.Errorf("not known as saved: error %v, %d reads, %d deletes; expected none", err, reads, deletes)
	}
}
