//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/plan"
)

func (this *builder) VisitCreateCredentialStore(stmt *algebra.CreateCredentialStore) (any, error) {
	return plan.NewQueryPlan(plan.NewCreateCredentialStore(stmt)), nil
}

func (this *builder) VisitAlterCredentialStore(stmt *algebra.AlterCredentialStore) (any, error) {
	return plan.NewQueryPlan(plan.NewAlterCredentialStore(stmt)), nil
}

func (this *builder) VisitDropCredentialStore(stmt *algebra.DropCredentialStore) (any, error) {
	return plan.NewQueryPlan(plan.NewDropCredentialStore(stmt)), nil
}
