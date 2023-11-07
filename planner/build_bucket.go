// Copyright 2023-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included
// in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
// in that file, in accordance with the Business Source License, use of this
// software will be governed by the Apache License, Version 2.0, included in
// the file licenses/APL2.txt.
package planner

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/plan"
)

func (this *builder) VisitCreateBucket(stmt *algebra.CreateBucket) (interface{}, error) {
	return plan.NewQueryPlan(plan.NewCreateBucket(stmt)), nil
}
func (this *builder) VisitAlterBucket(stmt *algebra.AlterBucket) (interface{}, error) {
	return plan.NewQueryPlan(plan.NewAlterBucket(stmt)), nil
}
func (this *builder) VisitDropBucket(stmt *algebra.DropBucket) (interface{}, error) {
	return plan.NewQueryPlan(plan.NewDropBucket(stmt)), nil
}
