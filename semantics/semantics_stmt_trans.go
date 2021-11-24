//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package semantics

import (
	"github.com/couchbase/query/algebra"
)

func (this *SemChecker) VisitStartTransaction(stmt *algebra.StartTransaction) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitCommitTransaction(stmt *algebra.CommitTransaction) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitRollbackTransaction(stmt *algebra.RollbackTransaction) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitTransactionIsolation(stmt *algebra.TransactionIsolation) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitSavepoint(stmt *algebra.Savepoint) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}
