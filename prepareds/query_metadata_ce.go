//  Copyright 2025-Present Couchbase, Inc.
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
)

func hasQueryMetadata(create bool, requestId, createReason string, waitOnCreate bool) (bool, errors.Error) {
	return false, errors.NewEnterpriseFeature("Query meta data bucket", "plan.has_query_metadata")
}

func PreparedsFromPersisted() {
	// no-op
}

func loadPrepared(name string) (*plan.Prepared, errors.Error) {
	return nil, errors.NewEnterpriseFeature("Plan Stability", "prepareds.load_prepared")
}

func deletePreparedPlans(adHocOnly bool) errors.Error {
	return errors.NewEnterpriseFeature("Plan Stability", "prepareds.delete_prepared_plans")
}
