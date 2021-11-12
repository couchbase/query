//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package rewrite

import (
	"github.com/couchbase/query/algebra"
)

func (this *Rewrite) VisitGrantRole(stmt *algebra.GrantRole) (interface{}, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitRevokeRole(stmt *algebra.RevokeRole) (interface{}, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitExplain(stmt *algebra.Explain) (interface{}, error) {
	return stmt.Statement().Accept(this)
}

func (this *Rewrite) VisitAdvise(stmt *algebra.Advise) (interface{}, error) {
	return stmt.Statement().Accept(this)
}

func (this *Rewrite) VisitPrepare(stmt *algebra.Prepare) (interface{}, error) {
	return stmt.Statement().Accept(this)
}

func (this *Rewrite) VisitExecute(stmt *algebra.Execute) (interface{}, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitInferKeyspace(stmt *algebra.InferKeyspace) (interface{}, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitInferExpression(stmt *algebra.InferExpression) (interface{}, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitUpdateStatistics(stmt *algebra.UpdateStatistics) (interface{}, error) {
	return stmt, stmt.MapExpressions(this)
}
