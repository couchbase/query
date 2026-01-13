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
	"github.com/couchbase/query/settings"
)

func (this *preparedCache) UpdatePlanStabilityMode(oldMode, newMode settings.PlanStabilityMode) errors.Error {
	if oldMode == newMode {
		return nil
	} else if oldMode == settings.PS_MODE_OFF &&
		(newMode == settings.PS_MODE_PREPARED_ONLY || newMode == settings.PS_MODE_AD_HOC) {
		// just turned on
		return persistPreparedStmts(newMode)
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
func persistPreparedStmts(newMode settings.PlanStabilityMode) errors.Error {
	// if newMode is AD_HOC, "downgrade" the mode to PREPARED_ONLY for prepared statements,
	// since the statements currently in the prepared cache are already explicitly prepared
	// (AD_HOC is used when an ad_hoc statement is implicitly prepared)
	if newMode == settings.PS_MODE_AD_HOC {
		newMode = settings.PS_MODE_PREPARED_ONLY
	}
	var err errors.Error
	PreparedsForeach(func(name string, ce *CacheEntry) bool {
		prepared := ce.Prepared
		if !prepared.Persist() {
			fullName := encodeName(prepared.Name(), prepared.QueryContext())
			prepared.SetPlanStabilityMode(newMode)
			// need to rebuild encoded plan since planStabilityMode changed
			encoded_plan, err1 := prepared.BuildEncodedPlan()
			if err1 != nil {
				err = errors.NewPreparedEncodedPlanError(fullName, err1)
				return false
			}
			err1 = dictionary.PersistPrepared(fullName, encoded_plan, false, newMode)
			if err1 != nil {
				err = errors.NewPreparedSavePlanError(fullName, err1)
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
			curMode := prepared.PlanStabilityMode()
			if newMode == settings.PS_MODE_OFF ||
				(curMode == settings.PS_MODE_AD_HOC && newMode == settings.PS_MODE_PREPARED_ONLY) {
				// delete the saved query plan
				err1 = dictionary.DeletePrepared(fullName)
				if err1 != nil {
					err = errors.NewPreparedDeletePlanError(fullName, err1)
					return false
				}
			}
			if curMode == settings.PS_MODE_AD_HOC {
				// newMode is OFF or PREPARED_ONLY, need to remove entry from cache
				// (saved query plan should be deleted above already)
				names = append(names, fullName)
			} else if curMode == settings.PS_MODE_PREPARED_ONLY && newMode == settings.PS_MODE_OFF {
				// newMode is OFF
				// (if newMode is AD_HOC, we do not change the plan stability mode,
				// since we implicitly downgrade AD_HOC to PREPARED_ONLY for
				// explicitly prepared statements)
				prepared.SetPlanStabilityMode(newMode)
				// need to rebuild encoded plan since planStabilityMode changed
				_, err1 = prepared.BuildEncodedPlan()
				if err1 != nil {
					err = errors.NewPreparedEncodedPlanError(fullName, err1)
					return false
				}
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
