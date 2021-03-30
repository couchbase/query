//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package semantics

import (
	"github.com/couchbase/query/algebra"
)

func (this *SemChecker) VisitCreateFunction(stmt *algebra.CreateFunction) (interface{}, error) {
	return nil, nil
}

func (this *SemChecker) VisitDropFunction(stmt *algebra.DropFunction) (interface{}, error) {
	return nil, nil
}

func (this *SemChecker) VisitExecuteFunction(stmt *algebra.ExecuteFunction) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}
