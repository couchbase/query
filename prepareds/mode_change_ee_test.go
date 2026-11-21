//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.
//
//go:build enterprise

package prepareds

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/settings"
)

// the saved plans, for the hooks of this package
type fakeSavedPlans struct {
	sync.Mutex
	mode       atomic.Int32 // the current plan stability mode
	saved      map[string]bool
	saves      []string
	deletes    []string
	deleteErr  map[string]errors.Error
	bulkErr    errors.Error
	bulkScans  []bool // adHocOnly of each scan
	saveHook   func(name string)
	eventOrder []string
}

func (this *fakeSavedPlans) event(e string) {
	this.Lock()
	this.eventOrder = append(this.eventOrder, e)
	this.Unlock()
}

func sorted(names []string) []string {
	rv := append([]string(nil), names...)
	sort.Strings(rv)
	return rv
}

// install the hooks, restored when the test ends
func stubSavedPlans(t *testing.T, mode settings.PlanStabilityMode) *fakeSavedPlans {
	fake := &fakeSavedPlans{saved: make(map[string]bool), deleteErr: make(map[string]errors.Error)}
	fake.mode.Store(int32(mode))

	m, s, d, b, h, bg, q := planStabilityModeFn, savePreparedFn, deleteSavedPlanFn, deletePreparedPlansFn,
		hasSavedPlanFn, inBackground, hasQueryMetadataFn
	t.Cleanup(func() {
		// the package-global state of the saved plans (see plan_load.go): entries saved with PREPARE SAVE, and
		// entries removed when not saved
		clearAllSaved()
		for _, e := range testEntries {
			clearUnloaded(e.name)
		}
		planStabilityModeFn, savePreparedFn, deleteSavedPlanFn, deletePreparedPlansFn, hasSavedPlanFn,
			inBackground, hasQueryMetadataFn = m, s, d, b, h, bg, q
	})

	planStabilityModeFn = func() settings.PlanStabilityMode { return settings.PlanStabilityMode(fake.mode.Load()) }
	savePreparedFn = func(prepared *plan.Prepared) errors.Error {
		name := encodeName(prepared.Name(), prepared.QueryContext())
		if fake.saveHook != nil {
			fake.saveHook(name)
		}
		fake.Lock()
		fake.saves = append(fake.saves, name)
		fake.saved[name] = true
		fake.eventOrder = append(fake.eventOrder, "save "+name)
		fake.Unlock()
		return nil
	}
	deleteSavedPlanFn = func(name string) errors.Error {
		fake.Lock()
		defer fake.Unlock()
		if err := fake.deleteErr[name]; err != nil {
			return err
		}
		fake.deletes = append(fake.deletes, name)
		delete(fake.saved, name)
		fake.eventOrder = append(fake.eventOrder, "delete "+name)
		return nil
	}
	deletePreparedPlansFn = func(adHocOnly bool) errors.Error {
		fake.Lock()
		defer fake.Unlock()
		fake.bulkScans = append(fake.bulkScans, adHocOnly)
		fake.eventOrder = append(fake.eventOrder, fmt.Sprintf("bulk %v", adHocOnly))
		return fake.bulkErr
	}
	hasSavedPlanFn = func(name string) (bool, errors.Error) {
		fake.Lock()
		defer fake.Unlock()
		return fake.saved[name], nil
	}
	inBackground = func(f func()) { f() }
	hasQueryMetadataFn = func(create bool, requestId, createReason string, waitOnCreate bool) (bool, errors.Error) {
		return true, nil
	}
	return fake
}

type testEntry struct {
	name                    string
	adHoc, inlineUdf, saved bool
}

var testEntries = []testEntry{
	{"adhoc1", true, false, false},
	{GetUdfPreparedName("default:b.s.f1"), false, true, false},
	{"p1", false, false, false},
	{"saved1", false, false, true}, // PREPARE SAVE
}

func newTestPrepared(e testEntry) *plan.Prepared {
	prepared := plan.NewPrepared(nil, nil, nil, nil, nil)
	prepared.SetName(e.name)
	prepared.SetText("SELECT 1")
	prepared.SetAdHoc(e.adHoc)
	prepared.SetAutoPrepared(e.adHoc)
	prepared.SetPersist(e.saved)
	if e.inlineUdf {
		prepared.SetInlineUdf()
	}
	return prepared
}

func cacheTestEntries() map[string]*plan.Prepared {
	// replaces the package-global prepared cache: fine for tests in this package, which run in sequence
	PreparedsInit(16)
	rv := make(map[string]*plan.Prepared, len(testEntries))
	for _, e := range testEntries {
		prepared := newTestPrepared(e)
		prepareds.add(prepared, false, false, nil)
		rv[e.name] = prepared
	}
	return rv
}

// MB-75517: a plan is saved by the current plan stability mode, not the mode the request started with; an ad hoc or
// inline UDF entry not saved is removed from the prepared cache
func TestPersistPreparedByCurrentMode(t *testing.T) {
	udf := GetUdfPreparedName("default:b.s.f1")
	for _, tc := range []struct {
		mode  settings.PlanStabilityMode
		saved []string // the others are not saved; ad hoc and inline UDF entries not saved are removed
	}{
		{settings.PS_MODE_OFF, []string{"saved1"}},
		{settings.PS_MODE_PREPARED_ONLY, []string{udf, "p1", "saved1"}},
		{settings.PS_MODE_AD_HOC, []string{"adhoc1", udf, "p1", "saved1"}},
		{settings.PS_MODE_AD_HOC_READ_ONLY, []string{"adhoc1", udf, "p1", "saved1"}},
	} {
		fake := stubSavedPlans(t, tc.mode)
		entries := cacheTestEntries()
		for _, e := range testEntries {
			if err := persistPrepared(entries[e.name]); err != nil {
				t.Errorf("%v: %s: error %v", tc.mode, e.name, err)
			}
		}
		if fmt.Sprint(sorted(fake.saves)) != fmt.Sprint(sorted(tc.saved)) {
			t.Errorf("%v: saved %v, expected %v", tc.mode, fake.saves, tc.saved)
		}
		for _, e := range testEntries {
			cached := prepareds.get(e.name, false) != nil
			expected := fake.saved[e.name] || !(e.adHoc || e.inlineUdf)
			if cached != expected {
				t.Errorf("%v: %s cached %v, expected %v", tc.mode, e.name, cached, expected)
			}
		}
	}
}

// MB-75517: a save decided under the old mode, in progress when the mode changes, completes before the mode change
// scans the prepared cache, which then deletes its saved plan
func TestModeChangeWaitsForSave(t *testing.T) {
	fake := stubSavedPlans(t, settings.PS_MODE_AD_HOC)
	entries := cacheTestEntries()

	entered := make(chan bool)
	release := make(chan bool)
	fake.saveHook = func(name string) {
		entered <- true
		<-release
	}
	saved := make(chan errors.Error)
	go func() {
		saved <- persistPrepared(entries["adhoc1"])
	}()
	<-entered
	fake.saveHook = nil

	// the mode is OFF: the cache scan waits for the save in progress
	fake.mode.Store(int32(settings.PS_MODE_OFF))
	updated := make(chan errors.Error)
	go func() {
		updated <- updatePreparedStmts(settings.PS_MODE_OFF, false)
	}()
	select {
	case err := <-updated:
		t.Fatalf("the mode change did not wait for the save in progress: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	if err := <-saved; err != nil {
		t.Errorf("save: %v", err)
	}
	if err := <-updated; err != nil {
		t.Errorf("mode change: %v", err)
	}
	if fake.saved["adhoc1"] || prepareds.get("adhoc1", false) != nil {
		t.Errorf("the plan saved under the old mode is left: saved %v, cached %v (events %v)", fake.saved["adhoc1"],
			prepareds.get("adhoc1", false) != nil, fake.eventOrder)
	}

	// a save after the mode changed follows the new mode
	fake.saves = nil
	prepareds.add(entries["adhoc1"], false, false, nil)
	if err := persistPrepared(entries["adhoc1"]); err != nil || len(fake.saves) != 0 {
		t.Errorf("save under the new mode: error %v, saves %v", err, fake.saves)
	}
}

// MB-75517: a mode change that deletes saved plans stops at the first failure, with the cache entries kept (the old
// mode is set back); otherwise the entries the new mode does not keep are removed, and the plans not in the cache
// deleted by the bulk scan
func TestUpdatePreparedStmts(t *testing.T) {
	udf := GetUdfPreparedName("default:b.s.f1")
	for _, tc := range []struct {
		newMode settings.PlanStabilityMode
		deleted []string
		adHoc   bool // the bulk scan
	}{
		{settings.PS_MODE_OFF, []string{"adhoc1", udf, "p1"}, false},
		{settings.PS_MODE_PREPARED_ONLY, []string{"adhoc1"}, true},
	} {
		for _, fail := range []bool{false, true} {
			fake := stubSavedPlans(t, tc.newMode)
			cacheTestEntries()
			if fail {
				fake.deleteErr["adhoc1"] = errors.NewQueryMetadataError("not available", nil)
			}
			err := updatePreparedStmts(tc.newMode, true)
			if fail {
				if err == nil || len(fake.bulkScans) != 0 || len(fake.deletes) >= len(tc.deleted) {
					t.Errorf("%v, fails: error %v, deleted %v, bulk scans %v; expected an error, stopped", tc.newMode,
						err, fake.deletes, fake.bulkScans)
				}
			} else if err != nil || fmt.Sprint(sorted(fake.deletes)) != fmt.Sprint(sorted(tc.deleted)) ||
				len(fake.bulkScans) != 1 || fake.bulkScans[0] != tc.adHoc {
				t.Errorf("%v: error %v, deleted %v, bulk scans %v; expected deleted %v, bulk scan ad hoc %v",
					tc.newMode, err, fake.deletes, fake.bulkScans, tc.deleted, tc.adHoc)
			}
			for _, e := range testEntries {
				cached := prepareds.get(e.name, false) != nil
				expected := fail || !(e.adHoc || e.inlineUdf)
				if cached != expected {
					t.Errorf("%v, fails %v: %s cached %v, expected %v", tc.newMode, fail, e.name, cached, expected)
				}
			}
		}
	}
}

// MB-75517: a mode change made on another node: this node deletes the saved plans of its entries the new mode does
// not use, or saves its entries not saved yet when plan stability is turned on
func TestApplyPlanStabilityModeSavedPlans(t *testing.T) {
	udf := GetUdfPreparedName("default:b.s.f1")
	for _, tc := range []struct {
		oldMode, newMode settings.PlanStabilityMode
		deleted, saves   []string
	}{
		{settings.PS_MODE_AD_HOC, settings.PS_MODE_OFF, []string{"adhoc1", udf, "p1", "p2"}, nil},
		{settings.PS_MODE_PREPARED_ONLY, settings.PS_MODE_OFF, []string{"adhoc1", udf, "p1", "p2"}, nil},
		{settings.PS_MODE_AD_HOC, settings.PS_MODE_PREPARED_ONLY, []string{"adhoc1"}, nil},
		{settings.PS_MODE_PREPARED_ONLY, settings.PS_MODE_AD_HOC_READ_ONLY, nil, nil},
		// p1 is saved; not "p2", saved already (e.g. by the node that made the change)
		{settings.PS_MODE_OFF, settings.PS_MODE_PREPARED_ONLY, nil, []string{"p1"}},
	} {
		fake := stubSavedPlans(t, tc.newMode)
		cacheTestEntries()
		p2 := newTestPrepared(testEntry{name: "p2"})
		prepareds.add(p2, false, false, nil)
		fake.saved["p2"] = true

		prepareds.ApplyPlanStabilityMode(tc.oldMode, tc.newMode)
		if fmt.Sprint(sorted(fake.deletes)) != fmt.Sprint(sorted(tc.deleted)) {
			t.Errorf("%v to %v: deleted %v, expected %v", tc.oldMode, tc.newMode, fake.deletes, tc.deleted)
		}
		if fmt.Sprint(sorted(fake.saves)) != fmt.Sprint(sorted(tc.saves)) {
			t.Errorf("%v to %v: saved %v, expected %v", tc.oldMode, tc.newMode, fake.saves, tc.saves)
		}
	}
}

// MB-75517: the deletes of saved plans for a mode change made on another node skip the plans the current mode keeps
// (e.g. plan stability turned on again), not to delete the plans saved again; a turn-on waits for a delete in
// progress
func TestUnusedSavedPlanDeletesStop(t *testing.T) {
	fake := stubSavedPlans(t, settings.PS_MODE_OFF)
	names := []savedPlanName{{"p1", false}, {"p2", false}, {"a1", true}, {"p3", false}}

	// plan stability turned on again during the first delete
	entered := make(chan bool)
	release := make(chan bool)
	first := true
	deleteSavedPlanFn = func(name string) errors.Error {
		if first {
			first = false
			entered <- true
			<-release
		}
		fake.Lock()
		fake.deletes = append(fake.deletes, name)
		fake.eventOrder = append(fake.eventOrder, "delete "+name)
		fake.Unlock()
		return nil
	}
	done := make(chan bool)
	go func() {
		deleteUnusedSavedPlans(settings.PS_MODE_PREPARED_ONLY, settings.PS_MODE_OFF, names)
		close(done)
	}()
	<-entered
	fake.mode.Store(int32(settings.PS_MODE_PREPARED_ONLY))
	barrier := make(chan bool)
	go func() {
		psModeBarrier()
		fake.event("turned on")
		close(barrier)
	}()
	select {
	case <-barrier:
		t.Fatalf("the turn-on did not wait for the delete in progress")
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	<-barrier
	<-done
	// PREPARED_ONLY does not keep the ad hoc plan: deleted
	if fmt.Sprint(fake.eventOrder) != "[delete p1 turned on delete a1]" {
		t.Errorf("events %v; expected p1 deleted before the turn-on, then only the ad hoc plan", fake.eventOrder)
	}
}

// MB-75517: a PREPARE that started with plan stability off is saved if it is on by the time the plan is cached; a
// plan added with AddPrepared() without plan stability (e.g. plan stability restore, which saves it itself) is not
func TestAddPreparedCurrentMode(t *testing.T) {
	fake := stubSavedPlans(t, settings.PS_MODE_PREPARED_ONLY)
	m := persistPreparedFn
	t.Cleanup(func() { persistPreparedFn = m })
	persistPreparedFn = persistPrepared
	PreparedsInit(16)

	if err := AddPrepared(newTestPrepared(testEntry{name: "restored1"}), false); err != nil || len(fake.saves) != 0 {
		t.Errorf("restored: error %v, saves %v; expected not saved", err, fake.saves)
	}
	if err := AddPreparedStatement(newTestPrepared(testEntry{name: "p1"}), false); err != nil ||
		fmt.Sprint(fake.saves) != "[p1]" {
		t.Errorf("PREPARE, plan stability turned on since: error %v, saves %v; expected p1 saved", err, fake.saves)
	}
}

// MB-75517: turning plan stability on saves the cached plans after the cache scan: not a plan removed or replaced in
// the cache since (the first save removes one of the other entries, and replaces the other)
func TestTurnOnSavesCachedPlansOnly(t *testing.T) {
	for _, remote := range []bool{false, true} {
		fake := stubSavedPlans(t, settings.PS_MODE_PREPARED_ONLY)
		m := persistPreparedFn
		t.Cleanup(func() { persistPreparedFn = m })
		persistPreparedFn = persistPrepared
		PreparedsInit(16)
		all := []string{"p1", "p2", "p3"}
		for _, name := range all {
			prepareds.add(newTestPrepared(testEntry{name: name}), false, false, nil)
		}

		var first string
		var removed, replaced string
		fake.saveHook = func(name string) {
			if first != "" {
				return
			}
			first = name
			var others []string
			for _, n := range all {
				if n != name {
					others = append(others, n)
				}
			}
			removed, replaced = others[0], others[1]
			deletePreparedFromCache(removed)
			prepareds.add(newTestPrepared(testEntry{name: replaced}), false, false, nil)
		}

		var err errors.Error
		if remote {
			// a turn-on made on another node
			saveUnsavedEntries(settings.PS_MODE_PREPARED_ONLY)
		} else {
			err = persistPreparedStmts(settings.PS_MODE_PREPARED_ONLY, "r1")
		}
		if err != nil || fmt.Sprint(fake.saves) != fmt.Sprint([]string{first}) {
			t.Errorf("remote %v: error %v, saves %v; expected only %s saved (%s removed, %s replaced)", remote,
				err, fake.saves, first, removed, replaced)
		}
	}
}
