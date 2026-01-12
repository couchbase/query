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
	"github.com/couchbase/query/settings"
)

func (this *preparedCache) UpdatePlanStabilityMode(oldMode, newMode settings.PlanStabilityMode) errors.Error {
	return errors.NewEnterpriseFeature("Plan Stability", "prepareds.update_plan_stability_mode")
}
