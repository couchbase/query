//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package rewrite

import (
	"github.com/couchbase/query/algebra"
)

func (this *Rewrite) VisitStartTransaction(stmt *algebra.StartTransaction) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitCommitTransaction(stmt *algebra.CommitTransaction) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitRollbackTransaction(stmt *algebra.RollbackTransaction) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitTransactionIsolation(stmt *algebra.TransactionIsolation) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitSavepoint(stmt *algebra.Savepoint) (any, error) {
	return stmt, stmt.MapExpressions(this)
}
