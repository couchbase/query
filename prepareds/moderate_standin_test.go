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
		gsiSyncedFn = s
	}(gsiSyncedFn)

	notFound := errors.NewUnrecognizedPreparedError(errors.NewCbIndexNotFoundError("ix1"))
	other := errors.NewUnrecognizedPreparedError(errors.NewPlanVerificationError("other", nil))

	gsiSyncedFn = func() bool { return false }
	if !isIndexerNotReady(notFound) || isIndexerNotReady(other) || isIndexerNotReady(nil) {
		t.Errorf("not synced: expected only an index not found to be not ready")
	}
	gsiSyncedFn = func() bool { return true }
	if isIndexerNotReady(notFound) || isIndexerNotReady(other) {
		t.Errorf("synced: expected nothing to be not ready")
	}
}
