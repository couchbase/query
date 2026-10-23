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

func (this *Rewrite) VisitCreatePrimaryIndex(stmt *algebra.CreatePrimaryIndex) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitCreateIndex(stmt *algebra.CreateIndex) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitDropIndex(stmt *algebra.DropIndex) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitAlterIndex(stmt *algebra.AlterIndex) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitBuildIndexes(stmt *algebra.BuildIndexes) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitCreateBucket(stmt *algebra.CreateBucket) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitAlterBucket(stmt *algebra.AlterBucket) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitDropBucket(stmt *algebra.DropBucket) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitCreateCatalog(stmt *algebra.CreateCatalog) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitAlterCatalog(stmt *algebra.AlterCatalog) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitDropCatalog(stmt *algebra.DropCatalog) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitCreateScope(stmt *algebra.CreateScope) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitDropScope(stmt *algebra.DropScope) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitCreateCollection(stmt *algebra.CreateCollection) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitAlterCollection(stmt *algebra.AlterCollection) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitDropCollection(stmt *algebra.DropCollection) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitFlushCollection(stmt *algebra.FlushCollection) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitCreateFunction(stmt *algebra.CreateFunction) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitDropFunction(stmt *algebra.DropFunction) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitExecuteFunction(stmt *algebra.ExecuteFunction) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitCreateSequence(stmt *algebra.CreateSequence) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitCreateKnowledge(stmt *algebra.CreateKnowledge) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitDropKnowledge(stmt *algebra.DropKnowledge) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitDropSequence(stmt *algebra.DropSequence) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitAlterSequence(stmt *algebra.AlterSequence) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitCreateCredentialStore(stmt *algebra.CreateCredentialStore) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitAlterCredentialStore(stmt *algebra.AlterCredentialStore) (any, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitDropCredentialStore(stmt *algebra.DropCredentialStore) (any, error) {
	return stmt, stmt.MapExpressions(this)
}
