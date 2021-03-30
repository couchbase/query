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

func (this *Rewrite) VisitCreatePrimaryIndex(stmt *algebra.CreatePrimaryIndex) (interface{}, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitCreateIndex(stmt *algebra.CreateIndex) (interface{}, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitDropIndex(stmt *algebra.DropIndex) (interface{}, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitAlterIndex(stmt *algebra.AlterIndex) (interface{}, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitBuildIndexes(stmt *algebra.BuildIndexes) (interface{}, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitCreateScope(stmt *algebra.CreateScope) (interface{}, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitDropScope(stmt *algebra.DropScope) (interface{}, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitCreateCollection(stmt *algebra.CreateCollection) (interface{}, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitDropCollection(stmt *algebra.DropCollection) (interface{}, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitFlushCollection(stmt *algebra.FlushCollection) (interface{}, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitCreateFunction(stmt *algebra.CreateFunction) (interface{}, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitDropFunction(stmt *algebra.DropFunction) (interface{}, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitExecuteFunction(stmt *algebra.ExecuteFunction) (interface{}, error) {
	return stmt, stmt.MapExpressions(this)
}
