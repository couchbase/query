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
	cacheFull := this.cache.Size() >= this.cache.Limit()
	if oldMode == newMode {
		return nil
	} else if oldMode == settings.PS_MODE_OFF &&
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
		return updatePreparedStmts(newMode, cacheFull)
	}
	// else, the following changes do not require changes to either prepared statements in
	// prepared cache, or saved prepared statements on disk:
	//  - changing from either PREPARED_ONLY or AD_HOC_READ_ONLY to AD_HOC
	//  - changing from either PREPARED_ONLY or AD_HOC to AD_HOC_READ_ONLY

	return nil
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
		if !prepared.Persist() {
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
		fullText := prepared.Text()
		if len(fullText) <= _TEXT_SIZE {
			text = fullText
		} else {
			text = fullText[:_TEXT_SIZE] + "..."
		}
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
	err := settings.SetDefaultPlanStabilitySetting(true)
	if err != nil {
		logging.Severef("Setting plan_stability to DEFAULT setting failed with error %v", err)
		panic(fmt.Sprintf("Error setting plan_stability to DEFAULT setting: %v", err))
	}

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

	for i := range names {
		logging.Warnf(fmt.Sprintf("QUERY_METADATA bucket is missing, removing prepared statement '%s'", names[i]))
		err := deletePreparedFromCache(names[i])
		if err != nil {
			logging.Errorf(fmt.Sprintf("Error encountered while removing prepared statement '%s': %v", names[i], err))
		}
	}

	queryMetadataCleanupCtrl.Lock()
	queryMetadataCleanupCtrl.state = _QUERY_METADATA_CLEANUP_INACTIVE
	queryMetadataCleanupCtrl.Unlock()
}
