//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.
//
//go:build !enterprise

package prepareds

import (
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/settings"
)

func hasQueryMetadata(create bool, requestId string, waitOnCreate bool) (bool, errors.Error) {
	return false, errors.NewEnterpriseFeature("Query meta data bucket", "plan.has_query_metadata")
}

func (this *preparedCache) UpdatePlanStabilityMode(oldMode, newMode settings.PlanStabilityMode, requestId string) errors.Error {
	return errors.NewEnterpriseFeature("Plan Stability", "prepareds.update_plan_stability_mode")
}

func persistPrepared(prepared *plan.Prepared) errors.Error {
	return errors.NewEnterpriseFeature("Plan Stability", "prepareds.update_plan_stability_mode")
}
