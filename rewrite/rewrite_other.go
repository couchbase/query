//  Copyright 2019-Present Couchbase, Inc.
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

func (this *Rewrite) VisitCreateUser(stmt *algebra.CreateUser) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitAlterUser(stmt *algebra.AlterUser) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitDropUser(stmt *algebra.DropUser) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitCreateGroup(stmt *algebra.CreateGroup) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitAlterGroup(stmt *algebra.AlterGroup) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitDropGroup(stmt *algebra.DropGroup) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitGrantRole(stmt *algebra.GrantRole) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitRevokeRole(stmt *algebra.RevokeRole) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitExplain(stmt *algebra.Explain) (any, error) {
	return stmt.Statement().Accept(this)
}

func (this *Rewrite) VisitExplainFunction(stmt *algebra.ExplainFunction) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitAdvise(stmt *algebra.Advise) (any, error) {
	return stmt.Statement().Accept(this)
}

func (this *Rewrite) VisitPrepare(stmt *algebra.Prepare) (any, error) {
	return stmt.Statement().Accept(this)
}

func (this *Rewrite) VisitExecute(stmt *algebra.Execute) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitInferKeyspace(stmt *algebra.InferKeyspace) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitInferExpression(stmt *algebra.InferExpression) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitUpdateStatistics(stmt *algebra.UpdateStatistics) (any, error) {
	return stmt, stmt.MapExpressions(this)
}
