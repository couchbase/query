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

// initialize cache from persisted entries
func PreparedsFromPersisted() {
	hasQueryMetadata, _ := dictionary.HasQueryMetadata(false, "", false)
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
		decodeFailedReason := make(map[string]errors.Error, _DEF_MAP_SIZE)
		decodeReprepReason := make(map[string]errors.Errors, _DEF_MAP_SIZE)
		success, fail, reprepare, err := dictionary.ForeachPreparedPlan(true, decodeFailedReason, decodeReprepReason, processPreparedPlan)

		preparedPrimeReport.Success = success
		preparedPrimeReport.Failed = fail
		preparedPrimeReport.Reprepared = reprepare

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

func processPreparedPlan(name, encoded_plan string, persist bool, decodeFailedReason map[string]errors.Error,
	decodeReprepReason map[string]errors.Errors) (success bool, reprep bool) {
	_, err, reprepareCause := DecodePrepared(name, encoded_plan, true,
		(settings.GetPlanStabilityMode() != settings.PS_MODE_OFF), logging.NULL_LOG)
	if err != nil {
		if decodeFailedReason != nil {
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

func loadPrepared(name string) (*plan.Prepared, errors.Error) {
	encoded_plan, err := dictionary.LoadPreparedPlan(name)
	if err != nil {
		return nil, err
	} else if encoded_plan == "" {
		return nil, nil
	}

	prepared, err, _ := DecodePrepared(name, encoded_plan, true,
		(settings.GetPlanStabilityMode() != settings.PS_MODE_OFF), logging.NULL_LOG)

	return prepared, err
}
