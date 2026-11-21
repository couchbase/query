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
	"sync"
	"time"
	"unicode/utf8"

	"github.com/couchbase/query-ee/dictionary"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/settings"
)

type queryMetadataCleanupState int

const (
	_QUERY_METADATA_CLEANUP_INACTIVE = queryMetadataCleanupState(iota)
	_QUERY_METADATA_CLEANUP_ACTIVE
)

type queryMetadataCleanup struct {
	sync.Mutex
	state queryMetadataCleanupState
}

var queryMetadataCleanupCtrl *queryMetadataCleanup

func planStabilityInit() {
	queryMetadataCleanupCtrl = &queryMetadataCleanup{
		state: _QUERY_METADATA_CLEANUP_INACTIVE,
	}
}

func (this *preparedCache) UpdatePlanStabilityMode(oldMode, newMode settings.PlanStabilityMode, requestId string) errors.Error {
	if oldMode == newMode {
		return nil
	}

	// MB-75479: ad hoc statements cached before AD_HOC are planned again, and saved, by their next execution
	if adoptsAdHocEntries(oldMode, newMode) {
		removeAutoPreparedEntries(oldMode, newMode)
	}

	if oldMode == settings.PS_MODE_OFF &&
		(newMode == settings.PS_MODE_PREPARED_ONLY || newMode == settings.PS_MODE_AD_HOC || newMode == settings.PS_MODE_AD_HOC_READ_ONLY) {
		// just turned on
		// MB-75517: deletes of saved plans for an earlier change to OFF, made on another node, that are in progress
		// complete first (see deleteUnusedSavedPlans()), so that they do not delete the plans saved now
		psModeBarrier()
		return persistPreparedStmts(newMode, requestId)
	} else if newMode == settings.PS_MODE_OFF &&
		(oldMode == settings.PS_MODE_PREPARED_ONLY || oldMode == settings.PS_MODE_AD_HOC || oldMode == settings.PS_MODE_AD_HOC_READ_ONLY) {
		// just turned off
		// to properly handle entries for inline UDFs, we explicitly set cacheFull to true if
		// plan stability is turned off, for situations where these entries may reside on disk
		// but not in the prepared cache yet (entry for inline UDFs only populated in the
		// prepared cache when it is first executed)
		return updatePreparedStmts(newMode, true)
	} else if newMode == settings.PS_MODE_PREPARED_ONLY &&
		(oldMode == settings.PS_MODE_AD_HOC || oldMode == settings.PS_MODE_AD_HOC_READ_ONLY) {
		// switch to PREPARED_ONLY from AD_HOC or AD_HOC_READ_ONLY
		// MB-75471: also delete the saved ad hoc plans of entries not in this node's prepared cache, e.g.
		// the ones prepared on other nodes
		return updatePreparedStmts(newMode, true)
	}
	// else, the following changes do not require changes to saved prepared statements on disk:
	//  - changing from either PREPARED_ONLY or AD_HOC_READ_ONLY to AD_HOC (ad hoc statements in the prepared
	//    cache were removed above)
	//  - changing from either PREPARED_ONLY or AD_HOC to AD_HOC_READ_ONLY

	return nil
}

// MB-75471: create QUERY_METADATA if it does not exist, and wait for it, before plan stability is turned on
func (this *preparedCache) EnsureQueryMetadata(requestId string) errors.Error {
	hasMetadata, err := hasQueryMetadataFn(true, requestId, "SAVE option of PREPARE or Plan Stability", true)
	if err != nil {
		return err
	} else if !hasMetadata {
		return errors.NewMissingQueryMetadataError("SAVE option of PREPARE or Plan Stability")
	}
	return waitQueryMetadataReads()
}

// test hooks
var hasQueryMetadataFn = hasQueryMetadata
var readQueryMetadataFn = readQueryMetadata
var readRetryInterval = 100 * time.Millisecond
var readMaxWait = 30 * time.Second

// a saved plan that does not exist
const _READ_CHECK_NAME = "__QUERY_METADATA_READ_CHECK__"

func readQueryMetadata() errors.Error {
	_, err := dictionary.LoadPreparedPlan(_READ_CHECK_NAME)
	return err
}

// MB-75506: QUERY_METADATA just created may not serve reads yet (e.g. its vbuckets are not ready): with plan
// stability on, a cache miss reads the saved plan, and fails the request if reading fails, rather than planning
// (and saving a new plan over a saved plan that could not be read). Wait until it serves reads before plan
// stability is turned on; if it does not in time, the mode change fails.
func waitQueryMetadataReads() errors.Error {
	interval := readRetryInterval
	deadline := time.Now().Add(readMaxWait)
	for {
		err := readQueryMetadataFn()
		if err == nil {
			return nil
		} else if !time.Now().Before(deadline) {
			logging.Errorf("QUERY_METADATA does not serve reads: %v", err)
			return err
		}
		time.Sleep(interval)
		if interval < time.Second {
			interval *= 2
		}
	}
}

// MB-75471: the mode changes from which ad hoc and inline UDF entries are removed from the prepared cache
// (see updatePreparedStmts())
func removesAdHocEntries(oldMode, newMode settings.PlanStabilityMode) bool {
	switch newMode {
	case settings.PS_MODE_OFF:
		return oldMode != settings.PS_MODE_OFF
	case settings.PS_MODE_PREPARED_ONLY:
		return oldMode == settings.PS_MODE_AD_HOC || oldMode == settings.PS_MODE_AD_HOC_READ_ONLY
	}
	return false
}

// MB-75479: the mode changes to AD_HOC: ad hoc statements cached before (auto prepared under another mode, and
// thus not marked as ad hoc) are not saved, and would keep being used as is
func adoptsAdHocEntries(oldMode, newMode settings.PlanStabilityMode) bool {
	return newMode == settings.PS_MODE_AD_HOC && oldMode != settings.PS_MODE_AD_HOC
}

// MB-75479: remove the ad hoc statements cached before the mode changed to AD_HOC (auto prepared, not marked as
// ad hoc, not saved with PREPARE SAVE) from the prepared cache: the next execution plans, and saves, them under
// AD_HOC
func removeAutoPreparedEntries(oldMode, newMode settings.PlanStabilityMode) {
	names := make([]string, 0, 128)
	PreparedsForeach(func(name string, ce *CacheEntry) bool {
		prepared := ce.Prepared
		if prepared.AutoPrepared() && !prepared.AdHoc() && !prepared.Persist() {
			names = append(names, encodeName(prepared.Name(), prepared.QueryContext()))
		}
		return true
	}, nil)
	for i := range names {
		// an entry may be gone by now
		deletePreparedFromCache(names[i])
	}
	if len(names) > 0 {
		logging.Infof("Plan stability mode changed from %v to %v: %d ad hoc statements removed from the prepared cache",
			oldMode, newMode, len(names))
	}
}

/*
 * MB-75471: a plan stability mode change made on another node, which updates the saved plans: remove the
 * entries the new mode does not keep from this node's prepared cache, same as updatePreparedStmts()
 *
 * MB-75517: the other node updates the saved plans when it makes the change, and this node may save plans under the
 * old mode until it gets the change (e.g. ad hoc plans under AD_HOC, or none under OFF). This node also updates the
 * saved plans of the entries in its prepared cache: when the mode change deletes saved plans, it deletes those of
 * the entries the new mode does not use; when plan stability is turned on, it saves the entries not saved yet (e.g.
 * prepared on this node during the change, or not distributed to the other node). In the background: not to hold up
 * the settings.
 */
func (this *preparedCache) ApplyPlanStabilityMode(oldMode, newMode settings.PlanStabilityMode) {
	if adoptsAdHocEntries(oldMode, newMode) {
		removeAutoPreparedEntries(oldMode, newMode)
	}
	if oldMode == settings.PS_MODE_OFF && newMode != settings.PS_MODE_OFF {
		inBackground(func() { saveUnsavedEntries(newMode) })
		return
	}
	if !removesAdHocEntries(oldMode, newMode) {
		return
	}

	// saves under the old mode in progress complete first, and their entries are found (see persistPrepared())
	psModeBarrier()
	names := make([]string, 0, 128)
	unused := make([]savedPlanName, 0, 128)
	PreparedsForeach(func(name string, ce *CacheEntry) bool {
		prepared := ce.Prepared
		if prepared.Persist() {
			return true
		}
		fullName := encodeName(prepared.Name(), prepared.QueryContext())
		if !keepsSavedPlan(newMode, false, prepared.AdHoc()) {
			unused = append(unused, savedPlanName{fullName, prepared.AdHoc()})
		}
		if prepared.AdHoc() || prepared.IsInlineUdf() {
			names = append(names, fullName)
		}
		return true
	}, nil)
	for i := range names {
		// an entry may be gone by now
		deletePreparedFromCache(names[i])
	}

	if len(names) > 0 {
		logging.Infof("Plan stability mode changed from %v to %v: %d entries removed from the prepared cache",
			oldMode, newMode, len(names))
	}
	if len(unused) > 0 {
		inBackground(func() { deleteUnusedSavedPlans(oldMode, newMode, unused) })
	}
}

// the name of a saved plan, and whether it is the plan of an ad hoc statement
type savedPlanName struct {
	name  string
	adHoc bool
}

// test hooks
var inBackground = func(f func()) { go f() }
var savePreparedFn = savePrepared
var deleteSavedPlanFn = func(name string) errors.Error { return dictionary.DeletePrepared(name) }
var deletePreparedPlansFn = deletePreparedPlans
var hasSavedPlanFn = func(name string) (bool, errors.Error) {
	encoded, err := dictionary.LoadPreparedPlan(name)
	return encoded != "", err
}

/*
 * MB-75517: delete the saved plans (if any) of this node's entries that the new mode does not use. Each delete
 * checks the current mode, under psModeLock as saves do: a plan the current mode keeps (the mode changed again since,
 * e.g. plan stability turned on again, and the plan saved again) is not deleted. A failure is logged; the saved plans
 * left are not used under the new mode (see usesSavedPlan()).
 */
func deleteUnusedSavedPlans(oldMode, newMode settings.PlanStabilityMode, names []savedPlanName) {
	var failed int
	var lastErr errors.Error
	for _, sp := range names {
		err := deleteUnusedSavedPlan(sp)
		if err != nil {
			if err.Code() == errors.E_MISSING_QUERY_METADATA {
				// nothing left to delete
				return
			}
			failed++
			lastErr = err
		}
	}
	if failed > 0 {
		logging.Errorf("Plan stability mode changed from %v to %v: deleting %d saved plans failed, e.g. %v",
			oldMode, newMode, failed, lastErr)
	}
}

// not if the current mode keeps the saved plan
func deleteUnusedSavedPlan(sp savedPlanName) errors.Error {
	psModeLock.RLock()
	defer psModeLock.RUnlock()
	if keepsSavedPlan(planStabilityModeFn(), false, sp.adHoc) {
		return nil
	}
	// deleting a plan that is not saved succeeds
	return deleteSavedPlanFn(sp.name)
}

// MB-75517: save the entries of this node's prepared cache that the new mode saves and are not saved yet (the node
// that turned plan stability on saves those of its own prepared cache); persistPrepared() does not save them if the
// mode changes again in the meantime
func saveUnsavedEntries(newMode settings.PlanStabilityMode) {
	// deletes in progress for an earlier change to OFF complete first, the ones after stop (see
	// deleteUnusedSavedPlans())
	psModeBarrier()
	entries := make([]*plan.Prepared, 0, 128)
	PreparedsForeach(func(name string, ce *CacheEntry) bool {
		if persistOnTurnOn(ce.Prepared, newMode) {
			entries = append(entries, ce.Prepared)
		}
		return true
	}, nil)
	var failed int
	var lastErr errors.Error
	for _, prepared := range entries {
		if !stillCached(prepared) {
			continue
		}
		fullName := encodeName(prepared.Name(), prepared.QueryContext())
		saved, err := hasSavedPlanFn(fullName)
		if err == nil && !saved {
			err = persistPreparedFn(prepared)
		}
		if err != nil {
			if err.Code() == errors.E_MISSING_QUERY_METADATA {
				return
			}
			failed++
			lastErr = err
		}
	}
	if failed > 0 {
		logging.Errorf("Plan stability turned on (%v): saving %d prepared statements failed, e.g. %v", newMode,
			failed, lastErr)
	}
}

// MB-75479: whether a cached plan is saved when plan stability is turned on: as when it is first added to the
// cache under the new mode (shouldPersist(); ad hoc statements are cached under OFF without being marked as ad
// hoc, and are not saved), and not a plan with a sequential scan kept local; not one saved already
func persistOnTurnOn(prepared *plan.Prepared, newMode settings.PlanStabilityMode) bool {
	return !prepared.Persist() && shouldPersist(prepared, newMode) && !seqScanKeptLocal(prepared)
}

/*
 * When plan stability mode changes from OFF to PREPARED_ONLY or AD_HOC or AD_HOC_READ_ONLY, need to
 * go through prepared cache and persist any prepared statement that's not already saved on disk.
 */
func persistPreparedStmts(newMode settings.PlanStabilityMode, requestId string) errors.Error {
	// check and create (if not exists) QUERY_METADATA bucket
	hasMetadata, err := hasQueryMetadataFn(true, requestId, "SAVE option of PREPARE or Plan Stability", true)
	if err != nil {
		return err
	} else if !hasMetadata {
		return errors.NewMissingQueryMetadataError("SAVE option of PREPARE or Plan Stability")
	}

	// MB-75517: the entries are saved after the cache scan, not while holding the cache locks (persistPrepared()
	// takes psModeLock, and may remove the entry from the cache)
	entries := make([]*plan.Prepared, 0, 128)
	PreparedsForeach(func(name string, ce *CacheEntry) bool {
		if persistOnTurnOn(ce.Prepared, newMode) {
			entries = append(entries, ce.Prepared)
		}
		return true
	}, nil)
	for _, prepared := range entries {
		if !stillCached(prepared) {
			continue
		}
		err = persistPrepared(prepared)
		if err != nil {
			return err
		}
	}
	return nil
}

// MB-75517: whether the cache entry still holds the plan, collected by a cache scan: a plan removed (e.g. deleted
// from system:prepareds) or replaced (e.g. PREPARE FORCE, saved under the new mode) since is not saved
func stillCached(prepared *plan.Prepared) bool {
	current := false
	prepareds.cache.Get(encodeName(prepared.Name(), prepared.QueryContext()), func(entry any) {
		current = entry.(*CacheEntry).Prepared == prepared
	})
	return current
}

/*
 * When plan stability mode changes from PREPARED_ONLY or AD_HOC or AD_HOC_READ_ONLY to OFF, need to
 * go through saved prepared plans and remove all that's not corresponding to a explicitly saved
 * prepared plan. When plan stability mode changes from AD_HOC or AD_HOC_READ_ONLY to PREPARED_ONLY,
 * need to go through prepared cache and remove saved prepared plan as necessary, and modify the
 * prepared statement.
 */
func updatePreparedStmts(newMode settings.PlanStabilityMode, cacheFull bool) errors.Error {
	// MB-75517: saves under the old mode in progress complete first, and their entries are found (see
	// persistPrepared()); the saved plans are deleted outside the cache scan
	psModeBarrier()
	unused := make([]string, 0, 128)
	names := make([]string, 0, 128)
	PreparedsForeach(func(name string, ce *CacheEntry) bool {
		prepared := ce.Prepared
		if !prepared.Persist() {
			fullName := encodeName(prepared.Name(), prepared.QueryContext())
			if !keepsSavedPlan(newMode, false, prepared.AdHoc()) {
				unused = append(unused, fullName)
			}
			if (prepared.AdHoc() || prepared.IsInlineUdf()) &&
				(newMode == settings.PS_MODE_OFF || newMode == settings.PS_MODE_PREPARED_ONLY) {
				// newMode is OFF or PREPARED_ONLY, need to remove entry from cache
				// (saved query plan should be deleted below)
				names = append(names, fullName)
			}
		}
		return true
	}, nil)

	// the first failure stops the change: the old mode is set back (see settings.updatePlanStabilitySetting()), with
	// the cache entries kept, but not the saved plans already deleted
	for _, name := range unused {
		err := deleteSavedPlanFn(name)
		if err != nil {
			if err.Code() == errors.E_MISSING_QUERY_METADATA {
				prepareds.HandleMissingQueryMetadata()
			}
			return errors.NewPreparedDeletePlanError(name, err)
		}
	}

	for i := range names {
		// an entry may be gone by now
		deletePreparedFromCache(names[i])
	}

	if cacheFull {
		// there may be entries on disk that's not currently in the prepareds cache
		return deletePreparedPlansFn(newMode == settings.PS_MODE_PREPARED_ONLY)
	}

	return nil
}

const _TEXT_SIZE = 50

// the first _TEXT_SIZE bytes of the text, followed by "..." if longer; MB-75478: not cutting a (UTF-8) character
func textPrefix(text string) string {
	if len(text) <= _TEXT_SIZE {
		return text
	}
	n := _TEXT_SIZE
	for n > 0 && !utf8.RuneStart(text[n]) {
		n--
	}
	return text[:n] + "..."
}

// MB-75517: a save of a plan (persistPrepared()), or a delete of a saved plan for a mode change
// (deleteUnusedSavedPlan()), checks the current mode and writes under the read lock; a mode change, once the new mode
// is in effect, waits for those in progress (psModeBarrier()): a save decided under the old mode completes before
// the cache scan of the change, which finds its entry; the ones after follow the new mode
var psModeLock sync.RWMutex

func psModeBarrier() {
	psModeLock.Lock()
	psModeLock.Unlock()
}

/*
 * MB-75517: whether the plan stability mode keeps a saved plan: under OFF only plans saved with PREPARE SAVE, and
 * under PREPARED_ONLY not the plans of ad hoc statements, i.e. not the saved plans a change to the mode deletes
 * (see updatePreparedStmts()).
 */
func keepsSavedPlan(mode settings.PlanStabilityMode, persist, adHoc bool) bool {
	switch mode {
	case settings.PS_MODE_OFF:
		return persist
	case settings.PS_MODE_PREPARED_ONLY:
		return persist || !adHoc
	}
	return true
}

/*
 * Save the plan. MB-75517: by the current plan stability mode: a request keeps the mode it started with, and a plan
 * that the mode no longer keeps (it changed since the request started) is not saved; an ad hoc or inline UDF entry
 * is then removed from the prepared cache, as the mode change does.
 */
func persistPrepared(prepared *plan.Prepared) errors.Error {
	psModeLock.RLock()
	defer psModeLock.RUnlock()
	if !keepsSavedPlan(planStabilityModeFn(), prepared.Persist(), prepared.AdHoc()) {
		if prepared.AdHoc() || prepared.IsInlineUdf() {
			removeUnsaved(prepared)
		}
		return nil
	}
	return savePreparedFn(prepared)
}

func savePrepared(prepared *plan.Prepared) errors.Error {
	var err1 error
	fullName := encodeName(prepared.Name(), prepared.QueryContext())
	encoded_plan := prepared.EncodedPlan()
	if encoded_plan == "" {
		encoded_plan, err1 = prepared.BuildEncodedPlan()
		if err1 != nil {
			return errors.NewPreparedEncodedPlanError(fullName, err1)
		}
	}
	// include the first _TEXT_SIZE bytes of the prepared text for ad hoc statements
	// (the entire text is part of encoded_plan, the separate text here is just for recognition)
	var text string
	if prepared.AdHoc() {
		text = textPrefix(prepared.Text())
	}
	err := dictionary.PersistPrepared(fullName, encoded_plan, text,
		prepared.Persist(), prepared.AdHoc(), prepared.IsInlineUdf(), prepared.GetKeyspaceReferences())
	if err != nil {
		if err.Code() == errors.E_MISSING_QUERY_METADATA {
			prepareds.HandleMissingQueryMetadata()
		}
		return errors.NewPreparedSavePlanError(fullName, err)
	}
	return nil
}

func deletePrepared(name string) errors.Error {
	err := dictionary.DeletePrepared(name)
	if err != nil {
		if err.Code() == errors.E_MISSING_QUERY_METADATA {
			prepareds.HandleMissingQueryMetadata()
		}
		return errors.NewPreparedDeletePlanError(name, err)
	}
	return nil
}

// when the QUERY_METADATA bucket is removed unexpected, do:
//   - reset plan stability to DEFAULT state (mode: OFF, error_policy: MODERATE)
//   - remove AD_HOC and INLINE_UDF entries from the prepared cache
//   - remove the SAVE marker from prepared statements (treat as regular prepared statement)
func (this *preparedCache) HandleMissingQueryMetadata() {
	ongoing := false
	queryMetadataCleanupCtrl.Lock()
	if queryMetadataCleanupCtrl.state == _QUERY_METADATA_CLEANUP_ACTIVE {
		ongoing = true
	} else {
		queryMetadataCleanupCtrl.state = _QUERY_METADATA_CLEANUP_ACTIVE
	}
	queryMetadataCleanupCtrl.Unlock()
	if ongoing {
		return
	}

	// do the cleanup in separate go routine
	go doQueryMetadataCleanup()
}

func doQueryMetadataCleanup() {

	logging.Infof("QUERY_METADATA bucket is missing, reset plan_stability to DEFAULT setting")

	// this node stops using plan stability right away; the setting is written to metakv below, unless it
	// was the default setting already (e.g. only plans saved with PREPARE SAVE)
	// MB-75508: not while a settings update on this node is in progress
	wasDefault := settings.ResetPlanStabilitySetting()

	names := make([]string, 0, 128)
	PreparedsForeach(func(name string, ce *CacheEntry) bool {
		prepared := ce.Prepared
		fullName := encodeName(prepared.Name(), prepared.QueryContext())
		if prepared.Persist() {
			logging.Warnf(fmt.Sprintf("QUERY_METADATA bucket is missing, reset SAVE option for prepared statement '%s'", fullName))
			prepared.SetPersist(false)
		} else if prepared.AdHoc() || prepared.IsInlineUdf() {
			names = append(names, fullName)
		}
		return true
	}, nil)

	// MB-75506: no plans saved with PREPARE SAVE any more
	clearAllSaved()

	for i := range names {
		logging.Warnf(fmt.Sprintf("QUERY_METADATA bucket is missing, removing prepared statement '%s'", names[i]))
		err := deletePreparedFromCache(names[i])
		if err != nil {
			logging.Errorf(fmt.Sprintf("Error encountered while removing prepared statement '%s': %v", names[i], err))
		}
	}

	// the cleanup stays active while retrying, so that further missing QUERY_METADATA errors don't start
	// another one
	if !wasDefault {
		persistDefaultPlanStabilitySetting()
	}

	queryMetadataCleanupCtrl.Lock()
	queryMetadataCleanupCtrl.state = _QUERY_METADATA_CLEANUP_INACTIVE
	queryMetadataCleanupCtrl.Unlock()
}

// test hooks
var persistDefaultSetting = settings.PersistDefaultPlanStabilitySetting
var isDefaultSetting = settings.IsDefaultPlanStabilitySetting
var resetRetryInterval = time.Second
var resetMaxRetryInterval = 30 * time.Second

/*
 * Write the default plan stability setting to metakv. A failure (e.g. metakv not available) is retried with
 * backoff until the write succeeds. The write does not replace settings changed in metakv since this node
 * last saw them (see settings.PersistDefaultPlanStabilitySetting()); retrying also stops if this node's
 * setting is no longer the default setting, i.e. this node already got such a change.
 */
func persistDefaultPlanStabilitySetting() {
	interval := resetRetryInterval
	for {
		if !isDefaultSetting() {
			logging.Infof("plan_stability setting changed since QUERY_METADATA bucket went missing, " +
				"DEFAULT setting not written")
			return
		}
		err := persistDefaultSetting()
		if err == nil {
			return
		}
		logging.Errorf("Writing plan_stability DEFAULT setting failed with error %v, retrying in %v", err, interval)
		time.Sleep(interval)
		interval *= 2
		if interval > resetMaxRetryInterval {
			interval = resetMaxRetryInterval
		}
	}
}
