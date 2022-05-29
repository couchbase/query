//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

var errBadFormat = fmt.Errorf("prepared must be an identifier or a string")

// Just a checker. The plan gets retrieved at request instantiation time, anyway
// and getting it here was just doubling effort, plus introduced a circular dependency
// VisitExecute is still part of the Visitor pattern because it will be used for
// the Execute syntax enhancements (MB-22574)
func (this *builder) VisitExecute(stmt *algebra.Execute) (interface{}, error) {
	val := stmt.PreparedValue()
	if val == nil || val.Type() != value.STRING {
		return nil, errors.NewUnrecognizedPreparedError(errBadFormat)
	}
	return plan.NewQueryPlan(plan.NewDiscard(OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL, OPT_COST_NOT_AVAIL)), nil
}
