//  Copyright 2019-Present Couchbase, Inc.
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

func (this *builder) VisitCreateFunction(stmt *algebra.CreateFunction) (interface{}, error) {
	return plan.NewQueryPlan(plan.NewCreateFunction(stmt)), nil
}

func (this *builder) VisitDropFunction(stmt *algebra.DropFunction) (interface{}, error) {
	return plan.NewQueryPlan(plan.NewDropFunction(stmt)), nil
}

func (this *builder) VisitExecuteFunction(stmt *algebra.ExecuteFunction) (interface{}, error) {
	return plan.NewQueryPlan(plan.NewExecuteFunction(stmt)), nil
}
