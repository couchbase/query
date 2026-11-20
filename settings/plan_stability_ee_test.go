//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.
//
//go:build enterprise

package settings

import (
	"fmt"
	"testing"
	"time"

	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
)

type fakePlanCache struct {
	updates      int
	modeAtCall   PlanStabilityMode
	updateErr    errors.Error
	applied      [][2]PlanStabilityMode
	ensures      int
	modeAtEnsure PlanStabilityMode
	ensureErr    errors.Error
}

func (this *fakePlanCache) EnsureQueryMetadata(requestId string) errors.Error {
	this.ensures++
	this.modeAtEnsure = GetPlanStabilityMode()
	return this.ensureErr
}

func (this *fakePlanCache) UpdatePlanStabilityMode(oldMode, newMode PlanStabilityMode, requestId string) errors.Error {
	this.updates++
	this.modeAtCall = GetPlanStabilityMode()
	return this.updateErr
}

func (this *fakePlanCache) ApplyPlanStabilityMode(oldMode, newMode PlanStabilityMode) {
	this.applied = append(this.applied, [2]PlanStabilityMode{oldMode, newMode})
}

func (this *fakePlanCache) DeleteUdfPrepared(udfName string, planStabilityMode PlanStabilityMode,
	planStabilityErrorPolicy PlanStabilityErrorPolicy) errors.Error {
	return nil
}

func setPlanStability(mode PlanStabilityMode, errorPolicy PlanStabilityErrorPolicy) {
	globalSettings = &querySettings{settings: map[string]any{
		PLAN_STABILITY: map[string]any{"mode": mode, "error_policy": errorPolicy},
	}}
}

// MB-75471: a plan stability setting update is validated before anything changes, the new mode is in effect
// when the prepared cache and the saved plans are updated, and a failure keeps the old setting
func TestUpdatePlanStabilitySetting(t *testing.T) {
	defer func(gs *querySettings, pc PlanCache) {
		globalSettings = gs
		planCache = pc
	}(globalSettings, planCache)

	fake := &fakePlanCache{}
	planCache = fake

	// an invalid field makes no change, whichever field is processed first
	for i := 0; i < 20; i++ {
		setPlanStability(PS_MODE_AD_HOC, PS_ERROR_FLEXIBLE)
		err := updatePlanStabilitySetting("r1", map[string]any{"mode": "off", "error_policy": "bogus"})
		if err == nil || fake.updates != 0 || GetPlanStabilityMode() != PS_MODE_AD_HOC ||
			GetPlanStabilityErrorPolicy() != PS_ERROR_FLEXIBLE {
			t.Fatalf("invalid error_policy: error %v, %d updates, mode %v, error policy %v", err, fake.updates,
				GetPlanStabilityMode(), GetPlanStabilityErrorPolicy())
		}
	}

	// turning plan stability on: QUERY_METADATA can not be created, nothing changes
	setPlanStability(PS_MODE_OFF, PS_ERROR_MODERATE)
	fake.ensureErr = errors.NewMissingQueryMetadataError("test")
	err := updatePlanStabilitySetting("r2", map[string]any{"mode": "ad_hoc", "error_policy": "strict"})
	if err == nil || fake.ensures != 1 || fake.updates != 0 || GetPlanStabilityMode() != PS_MODE_OFF {
		t.Errorf("no QUERY_METADATA: error %v, %d ensures, %d updates, mode %v", err, fake.ensures, fake.updates,
			GetPlanStabilityMode())
	}

	// turning plan stability on: QUERY_METADATA exists before the new mode is in effect, and the new mode
	// is in effect when the prepared cache and the saved plans are updated
	fake.ensureErr = nil
	fake.ensures = 0
	err = updatePlanStabilitySetting("r2", map[string]any{"mode": "ad_hoc", "error_policy": "strict"})
	if err != nil || fake.ensures != 1 || fake.modeAtEnsure != PS_MODE_OFF || fake.updates != 1 ||
		fake.modeAtCall != PS_MODE_AD_HOC || GetPlanStabilityMode() != PS_MODE_AD_HOC ||
		GetPlanStabilityErrorPolicy() != PS_ERROR_STRICT {
		t.Errorf("update: error %v, %d ensures, mode at ensure %v, %d updates, mode at update %v, mode %v, "+
			"error policy %v", err, fake.ensures, fake.modeAtEnsure, fake.updates, fake.modeAtCall,
			GetPlanStabilityMode(), GetPlanStabilityErrorPolicy())
	}

	// a field not present gets its default value (e.g. UNSET used); plan stability stays on, so
	// QUERY_METADATA is not checked again
	err = updatePlanStabilitySetting("r3", map[string]any{"mode": "prepared_only"})
	if err != nil || fake.ensures != 1 || GetPlanStabilityMode() != PS_MODE_PREPARED_ONLY ||
		GetPlanStabilityErrorPolicy() != PS_ERROR_DEFAULT {
		t.Errorf("update, error policy not present: error %v, %d ensures, mode %v, error policy %v", err,
			fake.ensures, GetPlanStabilityMode(), GetPlanStabilityErrorPolicy())
	}

	// updating the prepared cache and the saved plans fails: the old setting is kept
	fake.updateErr = errors.NewPreparedDeletePlanError("p1", fmt.Errorf("QUERY_METADATA not available"))
	err = updatePlanStabilitySetting("r4", map[string]any{"mode": "off", "error_policy": "flexible"})
	if err == nil || GetPlanStabilityMode() != PS_MODE_PREPARED_ONLY || GetPlanStabilityErrorPolicy() != PS_ERROR_DEFAULT {
		t.Errorf("update fails: error %v, mode %v, error policy %v", err, GetPlanStabilityMode(),
			GetPlanStabilityErrorPolicy())
	}
}

// MB-75471: a plan stability mode change made on another node updates this node's prepared cache
func TestProcessSettingsPlanStabilityMode(t *testing.T) {
	defer func(gs *querySettings, pc PlanCache) {
		globalSettings = gs
		planCache = pc
	}(globalSettings, planCache)

	fake := &fakePlanCache{}
	planCache = fake

	settings := func(mode PlanStabilityMode) []byte {
		return []byte(fmt.Sprintf(`{"plan_stability":{"mode":%d,"error_policy":1},"node":"n1"}`, mode))
	}

	setPlanStability(PS_MODE_AD_HOC, PS_ERROR_MODERATE)
	processSettings(settings(PS_MODE_PREPARED_ONLY), "n2", false)
	if len(fake.applied) != 1 || fake.applied[0] != [2]PlanStabilityMode{PS_MODE_AD_HOC, PS_MODE_PREPARED_ONLY} {
		t.Errorf("mode changed on another node: applied %v", fake.applied)
	}

	// mode not changed
	fake.applied = nil
	processSettings(settings(PS_MODE_PREPARED_ONLY), "n2", false)
	if len(fake.applied) != 0 {
		t.Errorf("mode not changed: applied %v", fake.applied)
	}

	// the initial settings
	processSettings(settings(PS_MODE_OFF), "", true)
	if len(fake.applied) != 0 || GetPlanStabilityMode() != PS_MODE_OFF {
		t.Errorf("initial settings: applied %v, mode %v", fake.applied, GetPlanStabilityMode())
	}

	// written by this node: not processed
	processSettings(settings(PS_MODE_AD_HOC), "n1", false)
	if len(fake.applied) != 0 || GetPlanStabilityMode() != PS_MODE_OFF {
		t.Errorf("this node's settings: applied %v, mode %v", fake.applied, GetPlanStabilityMode())
	}

	// standalone (no node name): not the initial settings (ApplyPlanStabilityMode() decides what to do
	// for each mode change)
	processSettings(settings(PS_MODE_AD_HOC), "", false)
	processSettings(settings(PS_MODE_OFF), "", false)
	if len(fake.applied) != 2 || fake.applied[0] != [2]PlanStabilityMode{PS_MODE_OFF, PS_MODE_AD_HOC} ||
		fake.applied[1] != [2]PlanStabilityMode{PS_MODE_AD_HOC, PS_MODE_OFF} {
		t.Errorf("standalone: applied %v", fake.applied)
	}
	fake.applied = nil
	setPlanStability(PS_MODE_OFF, PS_ERROR_MODERATE)
	processSettings(settings(PS_MODE_AD_HOC), "n1", false)
	if len(fake.applied) != 0 || GetPlanStabilityMode() != PS_MODE_OFF {
		t.Errorf("this node's settings: applied %v, mode %v", fake.applied, GetPlanStabilityMode())
	}
}

// MB-75471: if writing the new settings to metakv fails, the old settings are kept
func TestUpdatePersistSettings(t *testing.T) {
	defer func(gs *querySettings, pc PlanCache, s func([]byte, any) error) {
		globalSettings = gs
		planCache = pc
		setStoredSettings = s
		setUnconfirmedWrite(nil)
	}(globalSettings, planCache, setStoredSettings)

	planCache = &fakePlanCache{}
	var setErr error
	var written []byte
	setStoredSettings = func(val []byte, rev any) error {
		written = val
		return setErr
	}

	for _, fail := range []bool{false, true} {
		for _, persistPS := range []bool{false, true} {
			setPlanStability(PS_MODE_PREPARED_ONLY, PS_ERROR_MODERATE)
			setErr = nil
			if fail {
				setErr = fmt.Errorf("metakv not available")
			}
			written = nil

			var err errors.Error
			if persistPS {
				// restore
				err = UpdatePersistPlanStabilitySetting("r1", map[string]any{"mode": "ad_hoc", "error_policy": "strict"})
			} else {
				// UPDATE system:settings
				err, _ = UpdatePersistSettings(true, "r1",
					map[string]any{PLAN_STABILITY: map[string]any{"mode": "ad_hoc", "error_policy": "strict"}})
			}
			expectedMode, expectedPolicy := PS_MODE_AD_HOC, PS_ERROR_STRICT
			if fail {
				expectedMode, expectedPolicy = PS_MODE_PREPARED_ONLY, PS_ERROR_MODERATE
			}
			if (err != nil) != fail || written == nil || GetPlanStabilityMode() != expectedMode ||
				GetPlanStabilityErrorPolicy() != expectedPolicy {
				t.Errorf("write fails %v, restore %v: error %v, written %s, mode %v, error policy %v", fail,
					persistPS, err, written, GetPlanStabilityMode(), GetPlanStabilityErrorPolicy())
			}
		}
	}
}

// a node of a cluster
type namedNodeRemote struct {
	distributed.SystemRemoteAccess
}

func (this namedNodeRemote) WhoAmI() string {
	return "n1"
}

// MB-75508: resetting plan stability to the default setting (QUERY_METADATA missing) waits for a settings update
// in progress, which does not undo the reset when its write to metakv fails
func TestResetPlanStabilitySetting(t *testing.T) {
	defer func(gs *querySettings, pc PlanCache, s func([]byte, any) error) {
		globalSettings = gs
		planCache = pc
		setStoredSettings = s
		setUnconfirmedWrite(nil)
	}(globalSettings, planCache, setStoredSettings)

	planCache = &fakePlanCache{}
	for _, fail := range []bool{true, false} {
		setPlanStability(PS_MODE_PREPARED_ONLY, PS_ERROR_MODERATE)
		entered := make(chan bool)
		release := make(chan bool)
		setStoredSettings = func(val []byte, rev any) error {
			entered <- true
			<-release
			if fail {
				return fmt.Errorf("metakv not available")
			}
			return nil
		}

		// an update, writing its settings
		updated := make(chan errors.Error)
		go func() {
			err, _ := UpdatePersistSettings(true, "r1",
				map[string]any{PLAN_STABILITY: map[string]any{"mode": "ad_hoc"}})
			updated <- err
		}()
		<-entered

		// the reset waits for the update
		reset := make(chan bool)
		go func() {
			reset <- ResetPlanStabilitySetting()
		}()
		var wasDefault, resetEarly bool
		select {
		case wasDefault = <-reset:
			// the update still holds the lock (if any): released below, so that later tests are not blocked
			resetEarly = true
			t.Errorf("write fails %v: the reset did not wait for the update", fail)
		case <-time.After(100 * time.Millisecond):
		}

		close(release)
		if err := <-updated; (err != nil) != fail {
			t.Errorf("write fails %v: update error %v", fail, err)
		}
		if !resetEarly {
			wasDefault = <-reset
		}
		if wasDefault {
			t.Errorf("write fails %v: expected the setting not the default setting before the reset", fail)
		}
		if !IsDefaultPlanStabilitySetting() {
			t.Errorf("write fails %v: expected the default setting after the reset, got mode %v", fail,
				GetPlanStabilityMode())
		}
	}
}

// MB-75508: settings written by this node, whose write to metakv reported an error, which come back from metakv
// (they were written all the same) are processed, so that this node does not stay on the old settings
func TestUnconfirmedSettingsWrite(t *testing.T) {
	defer func(gs *querySettings, pc PlanCache, s func([]byte, any) error, ra distributed.SystemRemoteAccess) {
		globalSettings = gs
		planCache = pc
		setStoredSettings = s
		distributed.SetRemoteAccess(ra)
		setUnconfirmedWrite(nil)
	}(globalSettings, planCache, setStoredSettings, distributed.RemoteAccess())

	distributed.SetRemoteAccess(namedNodeRemote{distributed.RemoteAccess()})
	fake := &fakePlanCache{}
	planCache = fake
	var stored []byte
	var setErr error
	setStoredSettings = func(val []byte, rev any) error {
		stored = val
		return setErr
	}
	update := func(mode string) errors.Error {
		err, _ := UpdatePersistSettings(true, "r1", map[string]any{PLAN_STABILITY: map[string]any{"mode": mode}})
		return err
	}

	// the write is stored, but reports an error: this node keeps the old setting
	setPlanStability(PS_MODE_OFF, PS_ERROR_MODERATE)
	setErr = fmt.Errorf("timeout")
	if err := update("prepared only"); err == nil || GetPlanStabilityMode() != PS_MODE_OFF {
		t.Fatalf("expected the update to fail, and the old setting kept: %v, mode %v", err,
			GetPlanStabilityMode())
	}

	// the stored settings come back from metakv: processed
	fake.applied = nil
	processSettings(stored, "n1", false)
	if GetPlanStabilityMode() != PS_MODE_PREPARED_ONLY || len(fake.applied) != 1 ||
		fake.applied[0] != [2]PlanStabilityMode{PS_MODE_OFF, PS_MODE_PREPARED_ONLY} {
		t.Errorf("expected the stored setting applied: mode %v, applied %v", GetPlanStabilityMode(), fake.applied)
	}

	// again: skipped, as other settings of this node
	setPlanStability(PS_MODE_OFF, PS_ERROR_MODERATE)
	processSettings(stored, "n1", false)
	if GetPlanStabilityMode() != PS_MODE_OFF {
		t.Errorf("expected the stored setting skipped the second time, got mode %v", GetPlanStabilityMode())
	}

	// writes that succeed: the settings of an earlier write, which come back late, are skipped
	setErr = nil
	if err := update("ad_hoc"); err != nil {
		t.Fatalf("update: %v", err)
	}
	earlier := stored
	if err := update("prepared only"); err != nil {
		t.Fatalf("update: %v", err)
	}
	fake.applied = nil
	processSettings(earlier, "n1", false)
	if GetPlanStabilityMode() != PS_MODE_PREPARED_ONLY || len(fake.applied) != 0 {
		t.Errorf("expected earlier settings skipped: mode %v, applied %v", GetPlanStabilityMode(), fake.applied)
	}

	// a write that fails is followed by one that succeeds: the settings of the failed write are skipped
	setErr = fmt.Errorf("timeout")
	update("ad_hoc")
	failed := stored
	setErr = nil
	if err := update("ad_hoc_read_only"); err != nil {
		t.Fatalf("update: %v", err)
	}
	processSettings(failed, "n1", false)
	if GetPlanStabilityMode() != PS_MODE_AD_HOC_READ_ONLY {
		t.Errorf("expected the settings of the failed write skipped, got mode %v", GetPlanStabilityMode())
	}
}

// MB-75508: settings of this node that come back from metakv while their write is in progress are checked once
// the write completes: applied if the write reported an error, and not after a newer update on this node; and the
// settings of a write that succeeded are not applied later (e.g. after a reset)
func TestUnconfirmedSettingsWriteOrdering(t *testing.T) {
	defer func(gs *querySettings, pc PlanCache, s func([]byte, any) error, ra distributed.SystemRemoteAccess) {
		globalSettings = gs
		planCache = pc
		setStoredSettings = s
		distributed.SetRemoteAccess(ra)
		setUnconfirmedWrite(nil)
	}(globalSettings, planCache, setStoredSettings, distributed.RemoteAccess())

	distributed.SetRemoteAccess(namedNodeRemote{distributed.RemoteAccess()})
	fake := &fakePlanCache{}
	planCache = fake

	// the settings come back from metakv while the write is in progress (callback), and the write reports an
	// error (fail)
	var stored []byte
	var callback chan bool
	var fail bool
	setStoredSettings = func(val []byte, rev any) error {
		stored = val
		if callback != nil {
			done := callback
			go func() {
				processSettings(val, "n1", false)
				done <- true
			}()
			// let the settings coming back wait for the lock
			time.Sleep(10 * time.Millisecond)
		}
		if fail {
			return fmt.Errorf("timeout")
		}
		return nil
	}
	update := func(mode string, withCallback, writeFails bool) chan bool {
		if withCallback {
			callback = make(chan bool, 1)
		} else {
			callback = nil
		}
		fail = writeFails
		UpdatePersistSettings(true, "r1", map[string]any{PLAN_STABILITY: map[string]any{"mode": mode}})
		return callback
	}

	// the write reports an error: its settings, which came back during the write, are applied
	setPlanStability(PS_MODE_OFF, PS_ERROR_MODERATE)
	fake.applied = nil
	<-update("prepared only", true, true)
	if GetPlanStabilityMode() != PS_MODE_PREPARED_ONLY || len(fake.applied) != 1 {
		t.Errorf("write reports an error: mode %v, applied %v; expected prepared_only applied",
			GetPlanStabilityMode(), fake.applied)
	}

	// the write reports an error, and a newer update follows right away: the settings of the newer update stay
	for i := 0; i < 20; i++ {
		setPlanStability(PS_MODE_OFF, PS_ERROR_MODERATE)
		done := update("prepared only", true, true)
		update("ad_hoc", false, false)
		<-done
		if GetPlanStabilityMode() != PS_MODE_AD_HOC {
			t.Fatalf("run %d: newer update: mode %v; expected ad_hoc", i, GetPlanStabilityMode())
		}
	}

	// the write succeeds: its settings, coming back later (e.g. after a reset), are not applied
	setPlanStability(PS_MODE_OFF, PS_ERROR_MODERATE)
	update("ad_hoc", false, false)
	written := stored
	ResetPlanStabilitySetting()
	processSettings(written, "n1", false)
	if GetPlanStabilityMode() != PS_MODE_OFF {
		t.Errorf("write succeeded, then reset: mode %v; expected off", GetPlanStabilityMode())
	}
}
