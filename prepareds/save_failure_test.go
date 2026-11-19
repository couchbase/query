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
	"github.com/couchbase/query/parser/n1ql"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/settings"
)

// MB-75464: a plan that could not be saved is not kept in the cache, and a cache miss looks for the saved
// plan again
func TestSaveFailure(t *testing.T) {
	defer func(p func(*plan.Prepared) errors.Error) {
		persistPreparedFn = p
	}(persistPreparedFn)

	// replaces the package-global prepared cache: fine for tests in this package, which run in sequence
	PreparedsInit(16)

	var saveErr errors.Error
	saves := 0
	persistPreparedFn = func(*plan.Prepared) errors.Error {
		saves++
		return saveErr
	}
	failSave := func(fail bool) {
		saves = 0
		saveErr = nil
		if fail {
			saveErr = errors.NewPreparedSavePlanError("test", fmt.Errorf("QUERY_METADATA not available"))
		}
	}
	newPrepared := func(name, text string) *plan.Prepared {
		prepared := plan.NewPrepared(nil, nil, nil, nil, nil)
		prepared.SetName(name)
		prepared.SetText(text)
		return prepared
	}
	cached := func(name string) *plan.Prepared {
		if ce := prepareds.get(name, false); ce != nil {
			return ce.Prepared
		}
		return nil
	}

	// ad hoc statement, plan stability AD_HOC
	const adHocText = "SELECT 1 AS one FROM b USE KEYS \"x\""
	stmt, err := n1ql.ParseStatement2(adHocText, "default", "")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	for _, fail := range []bool{true, false} {
		failSave(fail)
		clearUnloaded("ap1")
		prepared := newPrepared("ap1", adHocText)
		prepared.SetAdHoc(true)
		added, er := AddAutoPreparePlan(stmt, prepared, settings.PS_MODE_AD_HOC)
		if er != nil || added == fail || saves != 1 {
			t.Errorf("ad hoc, save fails %v: added %v, error %v, %d saves", fail, added, er, saves)
		}
		if (cached("ap1") == prepared) == fail || isUnloaded("ap1") != fail {
			t.Errorf("ad hoc, save fails %v: cached %v, unloaded %v", fail, cached("ap1") == prepared,
				isUnloaded("ap1"))
		}
	}

	// inline UDF
	for _, fail := range []bool{true, false} {
		failSave(fail)
		name := GetUdfPreparedName("default:b.s.f1")
		clearUnloaded(name)
		prepared := newPrepared(name, "")
		prepared.SetInlineUdf()
		if er := AddUdfPrepared(prepared, false); er != nil || saves != 1 {
			t.Errorf("inline UDF, save fails %v: error %v, %d saves", fail, er, saves)
		}
		if (cached(name) == prepared) == fail || isUnloaded(name) != fail {
			t.Errorf("inline UDF, save fails %v: cached %v, unloaded %v", fail, cached(name) == prepared,
				isUnloaded(name))
		}
	}

	// prepared statement: a new name, and one replacing a cached plan for the same statement (PREPARE FORCE)
	const prepText = "SELECT 2 AS two FROM b USE KEYS \"y\""
	failSave(false)
	clearUnloaded("p1")
	old := newPrepared("p1", prepText)
	if er := addPrepared(old, true, true); er != nil || cached("p1") != old {
		t.Fatalf("prepared: error %v adding the first plan", er)
	}
	failSave(true)
	prepared := newPrepared("p1", prepText)
	if er := addPrepared(prepared, true, true); er == nil || saves != 1 {
		t.Errorf("prepared, replacing: expected error, got %v, %d saves", er, saves)
	}
	if cached("p1") != nil || !isUnloaded("p1") {
		t.Errorf("prepared, replacing: expected removed and unloaded, got cached %v, unloaded %v",
			cached("p1") != nil, isUnloaded("p1"))
	}

	failSave(true)
	clearUnloaded("p2")
	prepared = newPrepared("p2", prepText)
	if er := addPrepared(prepared, true, true); er == nil || cached("p2") != nil || !isUnloaded("p2") {
		t.Errorf("prepared, new: expected error, removed and unloaded, got %v, cached %v, unloaded %v", er,
			cached("p2") != nil, isUnloaded("p2"))
	}

	// a plan that replaced the one that failed to save is not removed
	failSave(false)
	clearUnloaded("p3")
	first := newPrepared("p3", prepText)
	second := newPrepared("p3", prepText)
	if er := addPrepared(second, true, true); er != nil {
		t.Fatalf("prepared: error %v adding the plan", er)
	}
	removeUnsaved(first)
	if cached("p3") != second {
		t.Errorf("prepared: removing a plan that is no longer cached removed the cached plan")
	}

	// a plan not saved (plan stability off, no SAVE) is cached without saving
	failSave(true)
	clearUnloaded("p4")
	prepared = newPrepared("p4", prepText)
	if er := addPrepared(prepared, false, true); er != nil || saves != 0 || cached("p4") != prepared {
		t.Errorf("prepared, not saved: error %v, %d saves, cached %v", er, saves, cached("p4") == prepared)
	}
}

// MB-75464: the saved plan is deleted before the cache entry, so that a failed delete can be retried
func TestDeleteFailure(t *testing.T) {
	defer func(d func(string) errors.Error) {
		deletePreparedFn = d
	}(deletePreparedFn)

	// replaces the package-global prepared cache: fine for tests in this package, which run in sequence
	PreparedsInit(16)

	var deleteErr errors.Error
	deletes := 0
	deletePreparedFn = func(string) errors.Error {
		deletes++
		return deleteErr
	}

	// a saved plan (SAVE): deleting it does not depend on the plan stability setting
	prepared := plan.NewPrepared(nil, nil, nil, nil, nil)
	prepared.SetName("d1")
	prepared.SetText("SELECT 3 AS three FROM b USE KEYS \"z\"")
	prepared.SetPersist(true)
	prepareds.add(prepared, false, false, nil)

	// not the caller's entry: nothing deleted
	err := DeletePreparedFunc("d1", func(*CacheEntry) bool { return false })
	if err == nil || err.Code() != errors.E_NO_SUCH_PREPARED || deletes != 0 || prepareds.get("d1", false) == nil {
		t.Errorf("not the caller's entry: error %v, %d deletes, cached %v", err, deletes,
			prepareds.get("d1", false) != nil)
	}

	// deleting the saved plan fails: the entry stays in the cache, and the delete can be retried
	deleteErr = errors.NewPreparedDeletePlanError("d1", fmt.Errorf("QUERY_METADATA not available"))
	err = DeletePreparedFunc("d1", nil)
	if err == nil || deletes != 1 || prepareds.get("d1", false) == nil {
		t.Errorf("delete fails: error %v, %d deletes, cached %v", err, deletes, prepareds.get("d1", false) != nil)
	}

	deleteErr = nil
	err = DeletePreparedFunc("d1", nil)
	if err != nil || deletes != 2 || prepareds.get("d1", false) != nil {
		t.Errorf("delete succeeds: error %v, %d deletes, cached %v", err, deletes, prepareds.get("d1", false) != nil)
	}

	// no longer in the cache
	err = DeletePreparedFunc("d1", nil)
	if err == nil || err.Code() != errors.E_NO_SUCH_PREPARED || deletes != 2 {
		t.Errorf("not cached: error %v, %d deletes", err, deletes)
	}

	// replaced (PREPARE FORCE) while deleting the saved plan: the new plan stays in the cache
	prepareds.add(prepared, false, false, nil)
	replacement := plan.NewPrepared(nil, nil, nil, nil, nil)
	replacement.SetName("d1")
	replacement.SetText(prepared.Text())
	replacement.SetPersist(true)
	deletePreparedFn = func(string) errors.Error {
		prepareds.add(replacement, false, false, nil)
		return nil
	}
	err = DeletePreparedFunc("d1", nil)
	if ce := prepareds.get("d1", false); err != nil || ce == nil || ce.Prepared != replacement {
		t.Errorf("replaced while deleting: error %v, replacement cached %v", err, ce != nil && ce.Prepared == replacement)
	}
}
