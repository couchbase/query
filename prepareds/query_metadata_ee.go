//  Copyright 2025-Present Couchbase, Inc.
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
	"time"

	json "github.com/couchbase/go_json"
	"github.com/couchbase/query-ee/dictionary"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/settings"
)

func hasQueryMetadata(create bool, requestId, createReason string, waitOnCreate bool) (bool, errors.Error) {
	return dictionary.HasQueryMetadata(create, requestId, createReason, waitOnCreate)
}

// initialize cache from persisted entries
func PreparedsFromPersisted() {
	hasQueryMetadata, _ := dictionary.HasQueryMetadata(false, "", "", false)
	if !hasQueryMetadata {
		return
	}

	preparedPrimeReport := &PrimeReport{
		StartTime: time.Now(),
	}

	store := datastore.GetDatastore()
	if store == nil {
		err := errors.NewNoDatastoreError()
		preparedPrimeReport.Reason = err.Error()
	} else {
		queryMetadata, err := store.GetQueryMetadata()
		if queryMetadata == nil {
			return
		}
		planStability := settings.IsPlanStabilityEnabled()
		planStabilityMode := settings.GetPlanStabilityMode()
		planStabilityErrorPolicy := settings.GetPlanStabilityErrorPolicy()
		decodeFailedReason := make(map[string]errors.Error, _DEF_MAP_SIZE)
		decodeReprepReason := make(map[string]errors.Errors, _DEF_MAP_SIZE)

		// MB-73680: entries deferred here (index-not-found while the GSI metadata watcher hasn't synced
		// yet, same race as PreparedsRemotePrime) get a delayed retry below instead of being counted as
		// a genuine failure.
		var pendingRetry []struct{ name, encoded_plan string }
		proc := func(name, encoded_plan string, persist, planStability bool, planStabilityMode settings.PlanStabilityMode,
			planStabilityErrorPolicy settings.PlanStabilityErrorPolicy, decodeFailedReason map[string]errors.Error,
			decodeReprepReason map[string]errors.Errors) (bool, bool) {
			return processPreparedPlan(name, encoded_plan, persist, planStability, planStabilityMode,
				planStabilityErrorPolicy, decodeFailedReason, decodeReprepReason, &pendingRetry)
		}
		success, fail, reprepare, err := dictionary.ForeachPreparedPlan(planStability, planStabilityMode,
			planStabilityErrorPolicy, decodeFailedReason, decodeReprepReason, proc)

		// one delayed retry for the entries deferred above before giving up on them; reprep=true as a
		// last-resort fallback if the watcher still hasn't synced. Each deferred entry was already
		// counted towards fail above (proc returned success=false for it), so a retry success here both
		// adds to success and backs out of fail.
		retried := len(pendingRetry)
		retrySuccess := 0
		if retried > 0 {
			time.Sleep(_PRIME_RETRY_DELAY)
			for _, p := range pendingRetry {
				_, decErr, reprepareCause := DecodePrepared(p.name, p.encoded_plan, true, false, planStability,
					planStabilityMode, planStabilityErrorPolicy, logging.NULL_LOG)
				if decErr == nil {
					retrySuccess++
					success++
					fail--
					if len(reprepareCause) > 0 {
						reprepare++
						decodeReprepReason[p.name] = reprepareCause
					}
				} else {
					decodeFailedReason[p.name] = decErr
				}
			}
		}

		preparedPrimeReport.Success = success
		preparedPrimeReport.Failed = fail
		preparedPrimeReport.Reprepared = reprepare
		preparedPrimeReport.Retried = retried
		preparedPrimeReport.RetrySuccess = retrySuccess

		if len(decodeFailedReason) > 0 {
			preparedPrimeReport.Reason = decodeFailedReason
		} else if err != nil {
			preparedPrimeReport.Reason = err.Error()
		}

		if len(decodeReprepReason) > 0 {
			preparedPrimeReport.RepreparedReason = decodeReprepReason
		}
	}

	preparedPrimeReport.EndTime = time.Now()

	if buf, err := json.Marshal(preparedPrimeReport); err == nil {
		logging.Infof("Prepared statement cache prime from persisted completed: %v", string(buf))
	}
}

// deferIndexerRace=true: an indexer-not-ready failure (see isIndexerNotReady) is appended to
// pendingRetry instead of being recorded as a genuine failure here - PreparedsFromPersisted gives it
// one delayed retry; anything else fails immediately, as this always has.
func processPreparedPlan(name, encoded_plan string, persist, planStability bool, planStabilityMode settings.PlanStabilityMode,
	planStabilityErrorPolicy settings.PlanStabilityErrorPolicy, decodeFailedReason map[string]errors.Error,
	decodeReprepReason map[string]errors.Errors, pendingRetry *[]struct{ name, encoded_plan string }) (success bool, reprep bool) {
	_, err, reprepareCause := DecodePreparedWithContext(name, "", encoded_plan, false, nil, true, false,
		planStability, true, planStabilityMode, planStabilityErrorPolicy, logging.NULL_LOG)
	if err != nil {
		if isIndexerNotReady(err) {
			*pendingRetry = append(*pendingRetry, struct{ name, encoded_plan string }{name, encoded_plan})
		} else if decodeFailedReason != nil {
			decodeFailedReason[name] = err
		}
	} else {
		success = true
		if len(reprepareCause) > 0 {
			reprep = true
			if decodeReprepReason != nil {
				decodeReprepReason[name] = reprepareCause
			}
		}
	}
	return
}

func loadPrepared(name string, planStabilityMode settings.PlanStabilityMode,
	planStabilityErrorPolicy settings.PlanStabilityErrorPolicy) (*plan.Prepared, errors.Error) {
	encoded_plan, err := dictionary.LoadPreparedPlan(name)
	if err != nil {
		return nil, err
	} else if encoded_plan == "" {
		return nil, nil
	}

	prepared, err, _ := DecodePrepared(name, encoded_plan, true, false, (planStabilityMode != settings.PS_MODE_OFF),
		planStabilityMode, planStabilityErrorPolicy, logging.NULL_LOG)

	return prepared, err
}

func deletePreparedPlans(adHocOnly bool) errors.Error {
	hasQueryMetadata, _ := dictionary.HasQueryMetadata(false, "", "", false)
	if !hasQueryMetadata {
		return nil
	}

	return dictionary.DeletePreparedPlans(adHocOnly)
}
