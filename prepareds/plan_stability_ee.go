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
 */
func (this *preparedCache) ApplyPlanStabilityMode(oldMode, newMode settings.PlanStabilityMode) {
	if adoptsAdHocEntries(oldMode, newMode) {
		removeAutoPreparedEntries(oldMode, newMode)
	}
	if !removesAdHocEntries(oldMode, newMode) {
		return
	}
	names := make([]string, 0, 128)
	PreparedsForeach(func(name string, ce *CacheEntry) bool {
		prepared := ce.Prepared
		if !prepared.Persist() && (prepared.AdHoc() || prepared.IsInlineUdf()) {
			names = append(names, encodeName(prepared.Name(), prepared.QueryContext()))
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
	hasMetadata, err := hasQueryMetadata(true, requestId, "SAVE option of PREPARE or Plan Stability", true)
	if err != nil {
		return err
	} else if !hasMetadata {
		return errors.NewMissingQueryMetadataError("SAVE option of PREPARE or Plan Stability")
	}

	PreparedsForeach(func(name string, ce *CacheEntry) bool {
		prepared := ce.Prepared
		if persistOnTurnOn(prepared, newMode) {
			err1 := persistPrepared(prepared)
			if err1 != nil {
				err = err1
				return false
			}
		}
		return true
	}, nil)
	return err
}

/*
 * When plan stability mode changes from PREPARED_ONLY or AD_HOC or AD_HOC_READ_ONLY to OFF, need to
 * go through saved prepared plans and remove all that's not corresponding to a explicitly saved
 * prepared plan. When plan stability mode changes from AD_HOC or AD_HOC_READ_ONLY to PREPARED_ONLY,
 * need to go through prepared cache and remove saved prepared plan as necessary, and modify the
 * prepared statement.
 */
func updatePreparedStmts(newMode settings.PlanStabilityMode, cacheFull bool) errors.Error {
	var err errors.Error
	missingQueryMetadata := false
	names := make([]string, 0, 128)
	PreparedsForeach(func(name string, ce *CacheEntry) bool {
		prepared := ce.Prepared
		if !prepared.Persist() {
			var err1 errors.Error
			fullName := encodeName(prepared.Name(), prepared.QueryContext())
			if newMode == settings.PS_MODE_OFF ||
				(prepared.AdHoc() && newMode == settings.PS_MODE_PREPARED_ONLY) {
				// delete the saved query plan
				err1 = dictionary.DeletePrepared(fullName)
				if err1 != nil {
					if err1.Code() == errors.E_MISSING_QUERY_METADATA {
						missingQueryMetadata = true
					}
					err = errors.NewPreparedDeletePlanError(fullName, err1)
					return false
				}
			}
			if (prepared.AdHoc() || prepared.IsInlineUdf()) &&
				(newMode == settings.PS_MODE_OFF || newMode == settings.PS_MODE_PREPARED_ONLY) {
				// newMode is OFF or PREPARED_ONLY, need to remove entry from cache
				// (saved query plan should be deleted above already)
				names = append(names, fullName)
			}
		}
		return true
	}, nil)
	if err != nil {
		if missingQueryMetadata {
			prepareds.HandleMissingQueryMetadata()
		}
		return err
	}

	for i := range names {
		err = deletePreparedFromCache(names[i])
		if err != nil {
			return err
		}
	}

	if cacheFull {
		// there may be entries on disk that's not currently in the prepareds cache
		err = deletePreparedPlans(newMode == settings.PS_MODE_PREPARED_ONLY)
		if err != nil {
			return err
		}
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

func persistPrepared(prepared *plan.Prepared) errors.Error {
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
