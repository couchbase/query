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
	"github.com/couchbase/query-ee/dictionary"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/settings"
)

func hasQueryMetadata(create bool, requestId string, waitOnCreate bool) (bool, errors.Error) {
	return dictionary.HasQueryMetadata(create, requestId, waitOnCreate)
}

func (this *preparedCache) UpdatePlanStabilityMode(oldMode, newMode settings.PlanStabilityMode, requestId string) errors.Error {
	if oldMode == newMode {
		return nil
	} else if oldMode == settings.PS_MODE_OFF &&
		(newMode == settings.PS_MODE_PREPARED_ONLY || newMode == settings.PS_MODE_AD_HOC) {
		// just turned on
		return persistPreparedStmts(newMode, requestId)
	} else if newMode == settings.PS_MODE_OFF &&
		(oldMode == settings.PS_MODE_PREPARED_ONLY || oldMode == settings.PS_MODE_AD_HOC) {
		// just turned off
		return updatePreparedStmts(newMode)
	} else {
		// switch between PREPARED_ONLY and AD_HOC
		return updatePreparedStmts(newMode)
	}

	return nil
}

/*
 * When plan stability mode changes from OFF to either PREPARED_ONLY or AD_HOC, need to go through
 * prepared cache and persist any prepared statement that's not already saved on disk.
 */
func persistPreparedStmts(newMode settings.PlanStabilityMode, requestId string) errors.Error {
	// check and create (if not exists) QUERY_METADATA bucket
	hasMetadata, err := hasQueryMetadata(true, requestId, true)
	if err == nil && !hasMetadata {
		err = errors.NewMissingQueryMetadataError("SAVE option of PREPARE or Plan Stability")
	}
	if err != nil {
		return err
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
 * When plan stability mode changes from either PREPARED_ONLY or AD_HOC to OFF, need to go through
 * saved prepared plans and remove all that's not corresponding to a explicitly saved prepared plan.
 * When plan stability mode changes between PREPARED_ONLY and AD_HOC, need to go through prepared
 * cache and remove saved prepared plan as necessary, and modify the prepared statement
 */
func updatePreparedStmts(newMode settings.PlanStabilityMode) errors.Error {
	var err errors.Error
	names := make([]string, 0, 128)
	PreparedsForeach(func(name string, ce *CacheEntry) bool {
		prepared := ce.Prepared
		if !prepared.Persist() {
			var err1 error
			fullName := encodeName(prepared.Name(), prepared.QueryContext())
			if newMode == settings.PS_MODE_OFF ||
				(prepared.AdHoc() && newMode == settings.PS_MODE_PREPARED_ONLY) {
				// delete the saved query plan
				err1 = dictionary.DeletePrepared(fullName)
				if err1 != nil {
					err = errors.NewPreparedDeletePlanError(fullName, err1)
					return false
				}
			}
			if prepared.AdHoc() {
				// newMode is OFF or PREPARED_ONLY, need to remove entry from cache
				// (saved query plan should be deleted above already)
				names = append(names, fullName)
			}
		}
		return true
	}, nil)
	if err != nil {
		return err
	}

	for i := range names {
		err = DeletePrepared(names[i])
		if err != nil {
			return err
		}
	}

	return nil
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
	err1 = dictionary.PersistPrepared(fullName, encoded_plan, prepared.Persist(), prepared.AdHoc())
	if err1 != nil {
		return errors.NewPreparedSavePlanError(fullName, err1)
	}
	return nil
}
