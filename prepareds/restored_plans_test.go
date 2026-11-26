//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package prepareds

import (
	"sync/atomic"
	"testing"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/settings"
	"github.com/couchbase/query/util"
)

// MB-75645: restored saved plans replace the cached plans under the same names, including a plan for a different
// statement, and the next use loads the restored plan whatever the plan stability mode
func TestEvictRestoredPlans(t *testing.T) {
	defer func(l func(string, settings.PlanStabilityMode, settings.PlanStabilityErrorPolicy) (*plan.Prepared,
		errors.Error)) {
		loadPreparedFn = l
	}(loadPreparedFn)

	// the package-global prepared cache and plan load state are put back as they were, so that the order of the
	// tests does not matter (e.g. TestCheckSavedPlan expects nothing unloaded)
	defer restorePlanLoadState(savePlanLoadState())
	defer func(cache *util.GenCache) {
		prepareds.cache = cache
	}(prepareds.cache)
	PreparedsInit(16)

	newPrepared := func(name, text string) *plan.Prepared {
		p := plan.NewPrepared(nil, nil, nil, nil, nil)
		p.SetName(name)
		p.SetText(text)
		return p
	}

	// cached: p1 for a different statement than the restored one, p2 not restored
	cached := newPrepared("p1", "PREPARE p1 FROM SELECT 1")
	prepareds.add(cached, false, false, nil)
	other := newPrepared("p2", "PREPARE p2 FROM SELECT 2")
	prepareds.add(other, false, false, nil)

	// the cache does not replace a plan for a different statement (MB-19509)
	restored := newPrepared("p1", "PREPARE p1 FROM SELECT 100")
	prepareds.add(restored, false, false, func(ce *CacheEntry) bool { return ce.Prepared.Text() == restored.Text() })
	if ce := prepareds.get("p1", false); ce == nil || ce.Prepared != cached {
		t.Fatalf("expected the cached plan for a different statement kept by the cache")
	}

	// p3: unloaded, with a stand-in for the replaced saved plan (MB-75507)
	markUnloaded("p3")
	planLoad.Lock()
	planLoad.standIns["p3"] = &standIn{}
	planLoad.Unlock()

	EvictRestoredPlans([]string{"p1", "p3"})

	if prepareds.get("p1", false) != nil {
		t.Errorf("expected the cached plan for p1 removed")
	}
	if ce := prepareds.get("p2", false); ce == nil || ce.Prepared != other {
		t.Errorf("expected the cached plan for p2 (not restored) kept")
	}
	if !isUnloaded("p1") || !isUnloaded("p3") || isUnloaded("p2") {
		t.Errorf("expected p1 and p3 marked unloaded, not p2: %v %v %v", isUnloaded("p1"), isUnloaded("p3"),
			isUnloaded("p2"))
	}
	planLoad.RLock()
	_, hasStandIn := planLoad.standIns["p3"]
	planLoad.RUnlock()
	if hasStandIn {
		t.Errorf("expected the stand-in for the replaced saved plan of p3 dropped")
	}

	// the next use of p1 loads the restored plan, with plan stability off as well
	loads := 0
	loadPreparedFn = func(name string, _ settings.PlanStabilityMode, _ settings.PlanStabilityErrorPolicy) (
		*plan.Prepared, errors.Error) {
		loads++
		if name == "p1" {
			return restored, nil
		}
		return nil, nil
	}
	for _, mode := range []settings.PlanStabilityMode{settings.PS_MODE_OFF, settings.PS_MODE_PREPARED_ONLY} {
		if !checkSavedPlan("p1", _LOOKUP_PREPARED, mode) {
			t.Errorf("mode %v: expected a cache miss for p1 to look for the saved plan", mode)
		}
	}
	loads = 0
	prepared, err := getPrepared("p1", "", nil, OPT_TRACK|OPT_VERIFY, nil, settings.PS_MODE_OFF,
		settings.PS_ERROR_MODERATE, nil, datastore.UNBOUNDED)
	if prepared != restored || err != nil || loads != 1 {
		t.Fatalf("plan stability off: got %v, %v, %d loads; expected the restored plan, 1 load", prepared, err, loads)
	}

	// nothing to evict
	EvictRestoredPlans(nil)
}

// the names tracked by the package-global planLoad (see plan_load.go)
type planLoadSnapshot struct {
	unloaded map[string]bool
	standIns map[string]*standIn
	saved    map[string]bool
}

func savePlanLoadState() planLoadSnapshot {
	planLoad.RLock()
	defer planLoad.RUnlock()
	rv := planLoadSnapshot{
		unloaded: make(map[string]bool, len(planLoad.unloaded)),
		standIns: make(map[string]*standIn, len(planLoad.standIns)),
		saved:    make(map[string]bool, len(planLoad.saved)),
	}
	for k, v := range planLoad.unloaded {
		rv.unloaded[k] = v
	}
	for k, v := range planLoad.standIns {
		rv.standIns[k] = v
	}
	for k, v := range planLoad.saved {
		rv.saved[k] = v
	}
	return rv
}

func restorePlanLoadState(s planLoadSnapshot) {
	planLoad.Lock()
	planLoad.unloaded = s.unloaded
	planLoad.standIns = s.standIns
	planLoad.saved = s.saved
	atomic.StoreInt32(&planLoad.unloadedSize, int32(len(s.unloaded)))
	atomic.StoreInt32(&planLoad.savedSize, int32(len(s.saved)))
	planLoad.Unlock()
}
