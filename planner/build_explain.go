//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package planner

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/plan"
)

func (this *builder) VisitExplain(stmt *algebra.Explain) (interface{}, error) {
	op, err := stmt.Statement().Accept(this)
	if err != nil {
		return nil, err
	}

	return plan.NewExplain(op.(plan.Operator), stmt.Text(), stmt.Statement().OptimHints()), nil
}
