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
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/planner"
	"github.com/couchbase/query/settings"
)

// MB-75507 test hooks: plans that fail verification, plans whose metadata changed, counted re-plans, and the GSI
// indexer of the keyspace of the statement
type standInHooks struct {
	failVerify map[*plan.Prepared]bool
	changed    map[*plan.Prepared]bool
	replanned  map[*plan.Prepared]bool
	verifies   int
	replans    int
	last       *plan.Prepared
	duringPlan func() // called while re-planning
	indexer    *fakeIndexer
	encoded    int // re-plans that built the encoded plan
}

type fakeIndexer struct {
	version uint64
}

func (this *fakeIndexer) Refresh() errors.Error {
	return nil
}

func (this *fakeIndexer) MetadataVersion() uint64 {
	return this.version
}

func setStandInHooks(t *testing.T) *standInHooks {
	h := &standInHooks{failVerify: make(map[*plan.Prepared]bool), changed: make(map[*plan.Prepared]bool),
		replanned: make(map[*plan.Prepared]bool), indexer: &fakeIndexer{version: 1}}
	v, m, r, k := verifyPreparedFn, metadataCheckFn, reprepareFn, keyspaceIndexersFn
	t.Cleanup(func() {
		verifyPreparedFn, metadataCheckFn, reprepareFn, keyspaceIndexersFn = v, m, r, k
	})
	keyspaceIndexersFn = func([]string) []indexerVersion {
		return []indexerVersion{{indexer: h.indexer, version: h.indexer.version}}
	}
	verifyPreparedFn = func(prepared *plan.Prepared) errors.Error {
		h.verifies++
		if h.failVerify[prepared] {
			return errors.NewPlanVerificationError("index not found", nil)
		}
		return nil
	}
	metadataCheckFn = func(prepared *plan.Prepared) bool {
		return !h.changed[prepared] && !h.failVerify[prepared]
	}
	reprepareFn = func(prepared *plan.Prepared, deltaKeyspaces map[string]bool, phaseTime *time.Duration,
		planStability bool, planStabilityMode settings.PlanStabilityMode,
		planStabilityErrorPolicy settings.PlanStabilityErrorPolicy, log logging.Log,
		scanConsistency datastore.ScanConsistency, encode bool) (*plan.Prepared, errors.Error) {
		h.replans++
		if encode {
			h.encoded++
		}
		if h.duringPlan != nil {
			h.duringPlan()
		}
		h.last = plan.NewPrepared(nil, nil, nil, nil, nil)
		h.last.SetName(prepared.Name())
		h.last.SetText(prepared.Text())
		h.last.SetPreparedTime(time.Now())
		h.replanned[h.last] = true
		return h.last, nil
	}
	return h
}

// MB-75507: under MODERATE, a cached plan that can not be used is re-planned once, and the new plan is used for
// the next executions while its metadata does not change
func TestModerateStandIn(t *testing.T) {
	h := setStandInHooks(t)

	// replaces the package-global prepared cache: fine for tests in this package, which run in sequence
	PreparedsInit(16)
	const name = "ms1"
	cached := plan.NewPrepared(nil, nil, nil, nil, nil)
	cached.SetName(name)
	cached.SetText("SELECT 1")
	cached.SetPreparedTime(time.Now())
	prepareds.add(cached, true, false, nil)
	h.failVerify[cached] = true

	get := func(policy settings.PlanStabilityErrorPolicy) (*plan.Prepared, errors.Error) {
		return getPrepared(name, "", nil, OPT_VERIFY, nil, settings.PS_MODE_PREPARED_ONLY, policy, nil,
			datastore.UNBOUNDED)
	}
	check := func(label string, prepared *plan.Prepared, err errors.Error, expected *plan.Prepared, replans int) {
		if err != nil || prepared != expected || h.replans != replans {
			t.Errorf("%s: got %p, %v, %d re-plans; expected %p, %d re-plans", label, prepared, err, h.replans,
				expected, replans)
		}
	}

	// the cached plan can not be used: re-planned, the new plan is kept
	prepared, err := get(settings.PS_ERROR_MODERATE)
	standInPlan := h.last
	check("first execution", prepared, err, standInPlan, 1)
	if ce := prepareds.get(name, false); ce == nil || ce.Prepared != cached || ce.standIn.Load() == nil ||
		ce.standIn.Load().prepared != standInPlan {
		t.Fatalf("expected the cached plan unchanged, and the new plan kept with it")
	}

	// the new plan does not replace the cached plan: its encoded plan is not built
	if h.encoded != 0 {
		t.Errorf("expected the encoded plan of the new plan not built")
	}

	// the next executions use the new plan: the cached plan is neither verified nor re-planned
	verifies := h.verifies
	for i := 0; i < 3; i++ {
		prepared, err = get(settings.PS_ERROR_MODERATE)
		check("next execution", prepared, err, standInPlan, 1)
	}
	if h.verifies != verifies {
		t.Errorf("expected no verification of the cached plan, got %d", h.verifies-verifies)
	}

	// not under another error policy: STRICT does not re-plan
	prepared, err = get(settings.PS_ERROR_STRICT)
	if err == nil || err.Code() != errors.E_REPREPARE || h.replans != 1 {
		t.Errorf("STRICT: got %v, %d re-plans; expected the STRICT error, no re-plan", err, h.replans)
	}

	// the metadata of the new plan changed: the cached plan is verified again, and re-planned
	h.changed[standInPlan] = true
	prepared, err = get(settings.PS_ERROR_MODERATE)
	check("metadata changed", prepared, err, h.last, 2)
	if prepared == standInPlan {
		t.Errorf("expected a new plan")
	}

	// an index of the keyspace changed (e.g. created again): the cached plan is verified again, even though the
	// metadata of the new plan did not change (e.g. a sequential scan)
	verifies = h.verifies
	h.indexer.version++
	prepared, err = get(settings.PS_ERROR_MODERATE)
	check("index changed", prepared, err, h.last, 3)
	if h.verifies == verifies {
		t.Errorf("expected the cached plan verified again")
	}

	// the cached plan can be used again (e.g. its index is back online): used, the new plan is dropped
	h.indexer.version++
	h.failVerify[cached] = false
	prepared, err = get(settings.PS_ERROR_MODERATE)
	check("cached plan usable", prepared, err, cached, 3)
	if ce := prepareds.get(name, false); ce.standIn.Load() != nil {
		t.Errorf("expected the new plan dropped")
	}

	// replacing the cached plan drops the new plan
	ce := prepareds.get(name, false)
	ce.standIn.Store(&standIn{prepared: h.last, standsFor: cached})
	replacement := plan.NewPrepared(nil, nil, nil, nil, nil)
	replacement.SetName(name)
	replacement.SetText("SELECT 1")
	prepareds.add(replacement, false, false, nil)
	if ce.standIn.Load() != nil {
		t.Errorf("expected the new plan dropped when the cached plan is replaced")
	}
}

// MB-75507: a plan re-planned for a cached plan that is replaced meanwhile (e.g. PREPARE FORCE, or a plan
// distributed by another node) does not stand in for the new cached plan
func TestModerateStandInReplaced(t *testing.T) {
	h := setStandInHooks(t)

	PreparedsInit(16)
	const name = "ms3"
	cached := plan.NewPrepared(nil, nil, nil, nil, nil)
	cached.SetName(name)
	cached.SetText("SELECT 1")
	cached.SetPreparedTime(time.Now())
	prepareds.add(cached, true, false, nil)
	h.failVerify[cached] = true

	replacement := plan.NewPrepared(nil, nil, nil, nil, nil)
	replacement.SetName(name)
	replacement.SetText("SELECT 1")
	replacement.SetPreparedTime(time.Now())
	h.duringPlan = func() {
		prepareds.add(replacement, false, false, nil)
	}

	get := func() (*plan.Prepared, errors.Error) {
		return getPrepared(name, "", nil, OPT_VERIFY, nil, settings.PS_MODE_PREPARED_ONLY,
			settings.PS_ERROR_MODERATE, nil, datastore.UNBOUNDED)
	}

	// the cached plan is replaced while it is re-planned: this execution uses the new plan
	prepared, err := get()
	if err != nil || prepared != h.last {
		t.Fatalf("first execution: got %p, %v; expected the re-planned plan", prepared, err)
	}
	h.duringPlan = nil

	// the next execution uses the new cached plan, which is verified, not the plan re-planned for the old one
	verifies := h.verifies
	prepared, err = get()
	if err != nil || prepared != replacement || h.replans != 1 || h.verifies == verifies {
		t.Errorf("next execution: got %p, %v, %d re-plans; expected the new cached plan %p, verified, 1 re-plan",
			prepared, err, h.replans, replacement)
	}
	if ce := prepareds.get(name, false); ce.standIn.Load() != nil {
		t.Errorf("expected the re-planned plan dropped")
	}
}

// MB-75507: an index of the keyspace that changes (e.g. created again) while the plan that can not be used is
// re-planned: the new plan does not stand in for it, as it is checked against the GSI indexer versions from
// before the re-plan
func TestModerateStandInIndexChangedWhileReplanning(t *testing.T) {
	h := setStandInHooks(t)

	PreparedsInit(16)
	const name = "ms4"
	defer clearUnloaded(name)

	// a cached plan
	cached := plan.NewPrepared(nil, nil, nil, nil, nil)
	cached.SetName(name)
	cached.SetText("SELECT 1")
	cached.SetPreparedTime(time.Now())
	prepareds.add(cached, true, false, nil)
	h.failVerify[cached] = true
	h.duringPlan = func() {
		h.indexer.version++
	}
	get := func() (*plan.Prepared, errors.Error) {
		return getPrepared(name, "", nil, OPT_VERIFY, nil, settings.PS_MODE_PREPARED_ONLY,
			settings.PS_ERROR_MODERATE, nil, datastore.UNBOUNDED)
	}
	if _, err := get(); err != nil || h.replans != 1 {
		t.Fatalf("first execution: %v, %d re-plans", err, h.replans)
	}
	h.duringPlan = nil
	verifies := h.verifies
	if _, err := get(); err != nil || h.verifies == verifies || h.replans != 2 {
		t.Errorf("cached: %v, %d re-plans; expected the cached plan verified again, and re-planned", err,
			h.replans)
	}

	// a saved plan, not in the cache, that can not be decoded
	PreparedsInit(16)
	encoded := undecodablePlan(name)
	h.duringPlan = func() {
		h.indexer.version++
	}
	if _, err, _ := decodePrepared(name, "", encoded, false, nil, true, false, true, false, true,
		settings.PS_MODE_PREPARED_ONLY, settings.PS_ERROR_MODERATE, logging.NULL_LOG); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if !isUnloaded(name) || getUnloadedStandIn(name, encoded) != nil {
		t.Errorf("unloaded: expected the saved plan unloaded, and the new plan not standing in for it")
	}
	clearUnloaded(name)

	// a saved plan, not in the cache, that is decoded but fails verification
	PreparedsInit(16)
	saved := plan.NewPrepared(plan.NewSequence(plan.NewDummyScan(0, 0, 0, 0)), nil, nil, nil, nil)
	saved.SetName(name)
	saved.SetText("SELECT 1")
	encoded, er := saved.BuildEncodedPlan()
	if er != nil {
		t.Fatalf("encoding: %v", er)
	}
	verifyPreparedFn = func(prepared *plan.Prepared) errors.Error {
		if h.replanned[prepared] {
			return nil
		}
		return errors.NewPlanVerificationError("index not found", nil)
	}
	h.duringPlan = func() {
		h.indexer.version++
	}
	if _, err, _ := decodePrepared(name, "", encoded, false, nil, true, false, true, false, true,
		settings.PS_MODE_PREPARED_ONLY, settings.PS_ERROR_MODERATE, logging.NULL_LOG); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if !isUnloaded(name) || getUnloadedStandIn(name, encoded) != nil {
		t.Errorf("decoded: expected the saved plan unloaded, and the new plan not standing in for it")
	}

	// without a change while re-planning, the new plan stands in for the saved plan
	h.duringPlan = nil
	if _, err, _ := decodePrepared(name, "", encoded, false, nil, true, false, true, false, true,
		settings.PS_MODE_PREPARED_ONLY, settings.PS_ERROR_MODERATE, logging.NULL_LOG); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if getUnloadedStandIn(name, encoded) != h.last {
		t.Errorf("decoded: expected the new plan standing in for the saved plan")
	}
}

// MB-75507: a PREPARE of a cached name (GetPlan()) returns the plan with its encoded plan, also a plan standing in
// for a cached plan that can not be used, which is built without it
func TestModerateStandInPrepare(t *testing.T) {
	h := setStandInHooks(t)

	PreparedsInit(16)
	const name = "ms5"
	cached := plan.NewPrepared(nil, nil, nil, nil, nil)
	cached.SetName(name)
	cached.SetText("SELECT 1")
	cached.SetPreparedTime(time.Now())
	prepareds.add(cached, true, false, nil)
	h.failVerify[cached] = true

	var prepContext planner.PrepareContext
	planner.NewPrepareContext(&prepContext, "req1", "", nil, nil, 0, 0, false, false, nil, nil, nil, true,
		settings.PS_MODE_PREPARED_ONLY, settings.PS_ERROR_MODERATE, datastore.UNBOUNDED)

	// re-planned for this PREPARE, and standing in for the cached plan for the next one
	for _, label := range []string{"re-planned", "standing in"} {
		prepared, err := prepareds.GetPlan(name, "SELECT 1", "", &prepContext)
		if err != nil || prepared != h.last || prepared.EncodedPlan() == "" {
			t.Errorf("%s: got %p, %v; expected the re-planned plan %p, with its encoded plan", label, prepared,
				err, h.last)
		}
	}
	if h.replans != 1 || h.encoded != 0 {
		t.Errorf("%d re-plans, %d with the encoded plan; expected 1, 0", h.replans, h.encoded)
	}
}

// MB-75507: under MODERATE, a saved plan that can not be used, and is not in the cache, is re-planned once, and
// the new plan is used for the next executions while the saved plan and its metadata do not change
func TestModerateUnloadedStandIn(t *testing.T) {
	h := setStandInHooks(t)

	PreparedsInit(16)
	const name = "ms2"
	defer clearUnloaded(name)

	// a plan that decodes without a datastore
	saved := plan.NewPrepared(plan.NewSequence(plan.NewDummyScan(0, 0, 0, 0)), nil, nil, nil, nil)
	saved.SetName(name)
	saved.SetText("SELECT 1")
	encoded, er := saved.BuildEncodedPlan()
	if er != nil {
		t.Fatalf("encoding: %v", er)
	}
	// the saved plan, as decoded, fails verification
	verifyPreparedFn = func(prepared *plan.Prepared) errors.Error {
		if h.replanned[prepared] {
			return nil
		}
		return errors.NewPlanVerificationError("index not found", nil)
	}

	load := func() *plan.Prepared {
		prepared, err, _ := decodePrepared(name, "", encoded, false, nil, true, false, true, false, true,
			settings.PS_MODE_PREPARED_ONLY, settings.PS_ERROR_MODERATE, logging.NULL_LOG)
		if err != nil {
			t.Fatalf("decoding: %v", err)
		}
		return prepared
	}

	// the saved plan is re-planned for this execution, not cached, and the new plan kept
	prepared := load()
	if prepared != h.last || prepareds.get(name, false) != nil || !isUnloaded(name) {
		t.Fatalf("expected the new plan, not cached, the saved plan unloaded")
	}
	if getUnloadedStandIn(name, encoded) != prepared {
		t.Errorf("expected the new plan used for the saved plan")
	}

	// the saved plan changed (e.g. on another node): not used, and dropped
	if getUnloadedStandIn(name, "other") != nil || getUnloadedStandIn(name, encoded) != nil {
		t.Errorf("expected the new plan not used for another saved plan, and dropped")
	}

	// the metadata of the new plan changed: not used
	prepared = load()
	h.changed[prepared] = true
	if getUnloadedStandIn(name, encoded) != nil {
		t.Errorf("expected the new plan not used once its metadata changed")
	}

	// an index of the keyspace changed (e.g. created again): not used
	prepared = load()
	h.indexer.version++
	if getUnloadedStandIn(name, encoded) != nil {
		t.Errorf("expected the new plan not used once an index of the keyspace changed")
	}

	// the name is no longer unloaded (e.g. cached, or nothing saved): dropped
	prepared = load()
	clearUnloaded(name)
	if getUnloadedStandIn(name, encoded) != nil {
		t.Errorf("expected the new plan dropped with the unloaded name")
	}
	if h.replans != 4 {
		t.Errorf("%d re-plans, expected 4", h.replans)
	}
}

// MB-75507: an index not found is a GSI metadata race only until the GSI metadata has synced
func TestIsIndexerNotReady(t *testing.T) {
	defer func(s func() bool) {
		gsiSettledFn = s
	}(gsiSettledFn)

	notFound := errors.NewUnrecognizedPreparedError(errors.NewCbIndexNotFoundError("ix1"))
	other := errors.NewUnrecognizedPreparedError(errors.NewPlanVerificationError("other", nil))

	gsiSettledFn = func() bool { return false }
	if !isIndexerNotReady(notFound) || isIndexerNotReady(other) || isIndexerNotReady(nil) {
		t.Errorf("not synced: expected only an index not found to be not ready")
	}
	gsiSettledFn = func() bool { return true }
	if isIndexerNotReady(notFound) || isIndexerNotReady(other) {
		t.Errorf("synced: expected nothing to be not ready")
	}
}

// MB-75516: a cached plan whose verification failed part way (having recorded the current metadata versions of the
// indexers verified before, so that its MetadataCheck() passes) is not used: it is verified in full again
func TestPartialVerification(t *testing.T) {
	for _, policy := range []settings.PlanStabilityErrorPolicy{settings.PS_ERROR_MODERATE, settings.PS_ERROR_STRICT} {
		h := setStandInHooks(t)
		PreparedsInit(16)
		const name = "pv1"
		cached := plan.NewPrepared(nil, nil, nil, nil, nil)
		cached.SetName(name)
		cached.SetText("SELECT 1")
		cached.SetPreparedTime(time.Now())
		prepareds.add(cached, true, false, nil)

		// the cached plan fails verification; its metadata check fails until it is verified, which records the
		// current versions (as a verification that fails part way)
		h.failVerify[cached] = true
		recorded := false
		verify := verifyPreparedFn
		verifyPreparedFn = func(prepared *plan.Prepared) errors.Error {
			if prepared == cached {
				recorded = true
			}
			return verify(prepared)
		}
		metadataCheckFn = func(prepared *plan.Prepared) bool {
			if prepared == cached {
				return recorded
			}
			return !h.changed[prepared]
		}

		get := func() (*plan.Prepared, errors.Error) {
			return getPrepared(name, "", nil, OPT_VERIFY, nil, settings.PS_MODE_PREPARED_ONLY, policy, nil,
				datastore.UNBOUNDED)
		}
		for i := 0; i < 2; i++ {
			prepared, err := get()
			if prepared == cached {
				t.Errorf("%v, execution %d: the cached plan that can not be used is used", policy, i)
			}
			if policy == settings.PS_ERROR_STRICT && (err == nil || err.Code() != errors.E_REPREPARE) {
				t.Errorf("%v, execution %d: got %v, expected the STRICT error", policy, i, err)
			}
			if policy == settings.PS_ERROR_MODERATE && (err != nil || prepared != h.last) {
				t.Errorf("%v, execution %d: got %p, %v; expected the re-planned plan %p", policy, i, prepared, err,
					h.last)
			}
		}
		if policy == settings.PS_ERROR_MODERATE && h.replans != 1 {
			t.Errorf("%v: %d re-plans, expected 1 (then the re-planned plan stands in)", policy, h.replans)
		}
	}
}

// MB-75516: a plan that verifies, whose prepared time was unset, is re-planned and replaced on request, whatever the
// error policy
func TestForcedReprepare(t *testing.T) {
	for _, policy := range []settings.PlanStabilityErrorPolicy{settings.PS_ERROR_MODERATE, settings.PS_ERROR_STRICT} {
		h := setStandInHooks(t)
		defer func(p func(*plan.Prepared) errors.Error) {
			persistPreparedFn = p
		}(persistPreparedFn)
		saves := 0
		persistPreparedFn = func(*plan.Prepared) errors.Error {
			saves++
			return nil
		}
		PreparedsInit(16)
		const name = "fr1"
		cached := plan.NewPrepared(nil, nil, nil, nil, nil)
		cached.SetName(name)
		cached.SetText("SELECT 1")
		cached.SetPreparedTime(time.Now())
		prepareds.add(cached, true, false, nil)
		cached.SetPreparedTime(time.Time{})

		prepared, err := getPrepared(name, "", nil, OPT_VERIFY, nil, settings.PS_MODE_PREPARED_ONLY, policy, nil,
			datastore.UNBOUNDED)
		ce := prepareds.get(name, false)
		if err != nil || prepared != h.last || h.replans != 1 || ce == nil || ce.Prepared != h.last || saves != 1 {
			t.Errorf("%v: got %p, %v, %d re-plans, %d saves; expected the re-planned plan %p, cached and saved",
				policy, prepared, err, h.replans, saves, h.last)
		}
	}
}

// MB-75516: with plan stability off, a plan saved with PREPARE SAVE, re-planned when it is loaded, is saved again
func TestSavedPlanReplannedOff(t *testing.T) {
	for _, persist := range []bool{true, false} {
		h := setStandInHooks(t)
		defer func(p func(*plan.Prepared) errors.Error) {
			persistPreparedFn = p
		}(persistPreparedFn)
		saves := 0
		persistPreparedFn = func(*plan.Prepared) errors.Error {
			saves++
			return nil
		}
		PreparedsInit(16)
		const name = "so1"
		// the unloaded and saved names are package-global: leave none for other tests
		defer clearUnloaded(name)
		defer clearAllSaved()

		saved := plan.NewPrepared(plan.NewSequence(plan.NewDummyScan(0, 0, 0, 0)), nil, nil, nil, nil)
		saved.SetName(name)
		saved.SetText("PREPARE SAVE so1 FROM SELECT 1")
		saved.SetPersist(persist)
		encoded, er := saved.BuildEncodedPlan()
		if er != nil {
			t.Fatalf("encoding: %v", er)
		}
		// the saved plan, as decoded, fails verification
		verifyPreparedFn = func(prepared *plan.Prepared) errors.Error {
			if h.replanned[prepared] {
				return nil
			}
			return errors.NewPlanVerificationError("index not found", nil)
		}
		reprepare := reprepareFn
		reprepareFn = func(prepared *plan.Prepared, deltaKeyspaces map[string]bool, phaseTime *time.Duration,
			planStability bool, planStabilityMode settings.PlanStabilityMode,
			planStabilityErrorPolicy settings.PlanStabilityErrorPolicy, log logging.Log,
			scanConsistency datastore.ScanConsistency, encode bool) (*plan.Prepared, errors.Error) {
			pl, err := reprepare(prepared, deltaKeyspaces, phaseTime, planStability, planStabilityMode,
				planStabilityErrorPolicy, log, scanConsistency, encode)
			pl.SetPersist(prepared.Persist())
			return pl, err
		}

		prepared, err, _ := decodePrepared(name, "", encoded, false, nil, true, false, false, false, true,
			settings.PS_MODE_OFF, settings.PS_ERROR_MODERATE, logging.NULL_LOG)
		expected := map[bool]int{true: 1, false: 0}[persist]
		if err != nil || prepared != h.last || saves != expected {
			t.Errorf("persist %v: got %v, %v, %d saves; expected the re-planned plan, %d saves", persist, prepared,
				err, saves, expected)
		}
	}
}

// MB-75516: an index not found is taken as not existing only once the GSI metadata has been synced for a while
func TestGsiSettled(t *testing.T) {
	defer SetGsiSettle(SetGsiSettle(time.Hour))
	datastore.SetGsiSynced()
	if gsiSettled() {
		t.Errorf("expected not settled right after the GSI metadata synced")
	}
	SetGsiSettle(0)
	if !gsiSettled() {
		t.Errorf("expected settled")
	}
}

// MB-75516: a saved plan that can not be decoded because an index is not found, while the GSI metadata is settling,
// is re-planned and the new plan cached, not saved; once settled, the saved plan is loaded again: used if the index
// is found by then, replaced by a new plan if it is not
func TestReplannedWhileSettling(t *testing.T) {
	for _, tc := range []struct {
		name       string
		mode       settings.PlanStabilityMode
		persist    bool // saved with PREPARE SAVE
		indexFound bool // the index is found once settled
	}{
		{"FLEXIBLE, index dropped", settings.PS_MODE_PREPARED_ONLY, false, false},
		{"FLEXIBLE, index found later", settings.PS_MODE_PREPARED_ONLY, false, true},
		{"off, PREPARE SAVE, index dropped", settings.PS_MODE_OFF, true, false},
		{"off, PREPARE SAVE, index found later", settings.PS_MODE_OFF, true, true},
	} {
		h := setStandInHooks(t)
		saves := 0
		loads := 0
		var settled, indexFound bool
		p, l, d, s := persistPreparedFn, loadPreparedFn, newPreparedFromEncodedFn, gsiSettledFn
		persistPreparedFn = func(*plan.Prepared) errors.Error {
			saves++
			return nil
		}
		gsiSettledFn = func() bool { return settled }
		const name = "rs1"
		text := "PREPARE rs1 FROM SELECT 1"
		if tc.persist {
			text = "PREPARE SAVE rs1 FROM SELECT 1"
		}
		saved := plan.NewPrepared(plan.NewSequence(plan.NewDummyScan(0, 0, 0, 0)), nil, nil, nil, nil)
		saved.SetName(name)
		saved.SetText(text)
		saved.SetPersist(tc.persist)
		saved.SetPreparedTime(time.Now())
		encoded, er := saved.BuildEncodedPlan()
		if er != nil {
			t.Fatalf("encoding: %v", er)
		}
		// decoding the saved plan: its index is not found (unless found)
		var decoded *plan.Prepared
		newPreparedFromEncodedFn = func(encoded string, remap bool, remapCache *plan.RemapCache) (*plan.Prepared, []byte, errors.Error) {
			if indexFound {
				decoded, _, err := d(encoded, remap, remapCache)
				return decoded, nil, err
			}
			partial := plan.NewPrepared(nil, nil, nil, nil, nil)
			partial.SetName(name)
			partial.SetPersist(tc.persist)
			return partial, []byte(`{"name":"rs1","text":"` + text + `"}`),
				errors.NewUnrecognizedPreparedError(errors.NewCbIndexNotFoundError("ix1"))
		}
		load := func() (*plan.Prepared, errors.Error) {
			prepared, err, _ := decodePrepared(name, "", encoded, false, nil, true, false,
				tc.mode != settings.PS_MODE_OFF, false, true, tc.mode, settings.PS_ERROR_FLEXIBLE, logging.NULL_LOG)
			decoded = prepared
			return prepared, err
		}
		loadPreparedFn = func(string, settings.PlanStabilityMode, settings.PlanStabilityErrorPolicy) (
			*plan.Prepared, errors.Error) {
			loads++
			return load()
		}
		// the re-planned plan keeps the SAVE option
		reprepare := reprepareFn
		reprepareFn = func(prepared *plan.Prepared, deltaKeyspaces map[string]bool, phaseTime *time.Duration,
			planStability bool, planStabilityMode settings.PlanStabilityMode,
			planStabilityErrorPolicy settings.PlanStabilityErrorPolicy, log logging.Log,
			scanConsistency datastore.ScanConsistency, encode bool) (*plan.Prepared, errors.Error) {
			pl, err := reprepare(prepared, deltaKeyspaces, phaseTime, planStability, planStabilityMode,
				planStabilityErrorPolicy, log, scanConsistency, encode)
			pl.SetPersist(prepared.Persist())
			return pl, err
		}
		PreparedsInit(16)

		// loaded while the GSI metadata is settling: re-planned, cached, not saved
		prepared, err := load()
		ce := prepareds.get(name, false)
		if err != nil || prepared != h.last || ce == nil || ce.Prepared != h.last || ce.provisional.Load() == 0 ||
			saves != 0 {
			t.Errorf("%s, settling: got %v, %v, %d saves; expected the re-planned plan cached, provisional, not "+
				"saved", tc.name, prepared, err, saves)
		}

		// settled: the next execution loads the saved plan again
		settled = true
		indexFound = tc.indexFound
		replanned := h.last
		prepared, err = getPrepared(name, "", nil, OPT_VERIFY, nil, tc.mode, settings.PS_ERROR_FLEXIBLE, nil,
			datastore.UNBOUNDED)
		ce = prepareds.get(name, false)
		expectedSaves := map[bool]int{true: 0, false: 1}[tc.indexFound]
		if err != nil || loads != 1 || prepared == replanned || prepared != decoded || ce == nil ||
			ce.Prepared != prepared || ce.provisional.Load() != 0 || saves != expectedSaves {
			t.Errorf("%s, settled: got %v, %v, %d loads, %d saves; expected the saved plan loaded again, %d saves",
				tc.name, prepared, err, loads, saves, expectedSaves)
		}

		// and then the cached plan is used
		if again, err := getPrepared(name, "", nil, OPT_VERIFY, nil, tc.mode, settings.PS_ERROR_FLEXIBLE, nil,
			datastore.UNBOUNDED); again != prepared || err != nil || loads != 1 {
			t.Errorf("%s, settled, again: got %v, %v, %d loads; expected the cached plan", tc.name, again, err, loads)
		}

		persistPreparedFn, loadPreparedFn, newPreparedFromEncodedFn, gsiSettledFn = p, l, d, s
		clearUnloaded(name)
		clearAllSaved()
	}
}

// MB-75516: once the GSI metadata has settled, a provisional cached plan is kept if the saved plan is not loaded
// again, there is no saved plan any more, or it can not be loaded (loaded again after _PROVISIONAL_RETRY)
func TestProvisionalKept(t *testing.T) {
	for _, tc := range []struct {
		name    string
		mode    settings.PlanStabilityMode
		policy  settings.PlanStabilityErrorPolicy
		persist bool         // saved with PREPARE SAVE
		loadErr errors.Error // loading the saved plan fails
		loads   int          // loads expected
		retry   bool         // loaded again after _PROVISIONAL_RETRY
	}{
		{"nothing saved", settings.PS_MODE_PREPARED_ONLY, settings.PS_ERROR_FLEXIBLE, false, nil, 1, false},
		{"off, nothing saved", settings.PS_MODE_OFF, settings.PS_ERROR_FLEXIBLE, true, nil, 1, false},
		{"load fails", settings.PS_MODE_PREPARED_ONLY, settings.PS_ERROR_FLEXIBLE, false,
			errors.NewPlanVerificationError("load failed", nil), 1, true},
		{"STRICT", settings.PS_MODE_PREPARED_ONLY, settings.PS_ERROR_STRICT, false, nil, 0, false},
		{"MODERATE", settings.PS_MODE_PREPARED_ONLY, settings.PS_ERROR_MODERATE, false, nil, 0, false},
		{"off, not PREPARE SAVE", settings.PS_MODE_OFF, settings.PS_ERROR_FLEXIBLE, false, nil, 0, false},
	} {
		setStandInHooks(t)
		loads := 0
		l, s := loadPreparedFn, gsiSettledFn
		gsiSettledFn = func() bool { return true }
		loadPreparedFn = func(string, settings.PlanStabilityMode, settings.PlanStabilityErrorPolicy) (
			*plan.Prepared, errors.Error) {
			loads++
			return nil, tc.loadErr
		}
		PreparedsInit(16)
		const name = "pk1"
		cached := plan.NewPrepared(nil, nil, nil, nil, nil)
		cached.SetName(name)
		cached.SetText("PREPARE pk1 FROM SELECT 1")
		cached.SetPersist(tc.persist)
		cached.SetPreparedTime(time.Now())
		prepareds.add(cached, true, false, nil)
		ce := prepareds.get(name, false)
		ce.provisional.Store(1)
		get := func(label string, loadsExpected int) {
			prepared, err := getPrepared(name, "", nil, OPT_VERIFY, nil, tc.mode, tc.policy, nil,
				datastore.UNBOUNDED)
			if prepared != cached || err != nil || loads != loadsExpected {
				t.Errorf("%s, %s: got %v, %v, %d loads; expected the cached plan, %d loads", tc.name, label,
					prepared, err, loads, loadsExpected)
			}
		}

		get("settled", tc.loads)
		if at := ce.provisional.Load(); (at > time.Now().UnixNano()) != tc.retry || (at == 0) == tc.retry {
			t.Errorf("%s: provisional %v; expected a retry later: %v", tc.name, at, tc.retry)
		}
		// not loaded again before _PROVISIONAL_RETRY
		get("again", tc.loads)
		if tc.retry {
			ce.provisional.Store(1) // _PROVISIONAL_RETRY has passed
			get("retry", tc.loads+1)
		}

		loadPreparedFn, gsiSettledFn = l, s
		clearUnloaded(name)
		clearAllSaved()
	}
}

// MB-75516: the plan loaded again for a provisional cached plan replaces it, and is checked as a cached plan
// (e.g. verified if its metadata changed); one not kept in the cache is used as loaded
func TestProvisionalReloaded(t *testing.T) {
	for _, cached := range []bool{true, false} {
		h := setStandInHooks(t)
		l, s := loadPreparedFn, gsiSettledFn
		gsiSettledFn = func() bool { return true }
		PreparedsInit(16)
		const name = "pr1"
		newPlan := func() *plan.Prepared {
			p := plan.NewPrepared(nil, nil, nil, nil, nil)
			p.SetName(name)
			p.SetText("PREPARE pr1 FROM SELECT 1")
			p.SetPreparedTime(time.Now())
			return p
		}
		prepareds.add(newPlan(), true, false, nil)
		prepareds.get(name, false).provisional.Store(1)
		loaded := newPlan()
		loadPreparedFn = func(string, settings.PlanStabilityMode, settings.PlanStabilityErrorPolicy) (
			*plan.Prepared, errors.Error) {
			if cached {
				prepareds.add(loaded, true, false, nil)
			} else {
				prepareds.cache.Delete(name, nil) // e.g. it could not be saved (see removeUnsaved())
			}
			return loaded, nil
		}
		// the metadata of the plan loaded has changed since
		h.changed[loaded] = true

		prepared, err := getPrepared(name, "", nil, OPT_VERIFY, nil, settings.PS_MODE_PREPARED_ONLY,
			settings.PS_ERROR_FLEXIBLE, nil, datastore.UNBOUNDED)
		expectedVerifies := map[bool]int{true: 1, false: 0}[cached]
		if prepared != loaded || err != nil || h.verifies != expectedVerifies {
			t.Errorf("cached %v: got %v, %v, %d verifications; expected the plan loaded, %d verifications", cached,
				prepared, err, h.verifies, expectedVerifies)
		}

		loadPreparedFn, gsiSettledFn = l, s
		clearUnloaded(name)
		clearAllSaved()
	}
}
