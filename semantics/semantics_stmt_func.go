//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package semantics

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/expression"
)

func (this *SemChecker) VisitCreateFunction(stmt *algebra.CreateFunction) (interface{}, error) {

	// this code cannot go in algebra, functions or functions/inline because of circular references
	// between algebra, expression and functions
	body, ok := stmt.Body().(interface{ Expressions() expression.Expressions })
	if ok {
		exprs := body.Expressions()
		if len(exprs) > 0 {
			return nil, exprs.MapExpressions(this)
		}
	}
	return nil, nil
}

func (this *SemChecker) VisitDropFunction(stmt *algebra.DropFunction) (interface{}, error) {
	return nil, nil
}

func (this *SemChecker) VisitExecuteFunction(stmt *algebra.ExecuteFunction) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}
