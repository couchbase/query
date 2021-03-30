//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package semantics

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
)

func (this *SemChecker) VisitCreatePrimaryIndex(stmt *algebra.CreatePrimaryIndex) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitCreateIndex(stmt *algebra.CreateIndex) (interface{}, error) {
	gsi := stmt.Using() == datastore.GSI || stmt.Using() == datastore.DEFAULT
	for i, term := range stmt.Keys() {
		if term.HasAttribute(algebra.IK_MISSING) && (i > 0 || !gsi) {
			return nil, errors.NewSemanticsError(nil, "MISSING attribute only allowed on GSI index leading key")
		}
	}
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitDropIndex(stmt *algebra.DropIndex) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitAlterIndex(stmt *algebra.AlterIndex) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitBuildIndexes(stmt *algebra.BuildIndexes) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitCreateScope(stmt *algebra.CreateScope) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitDropScope(stmt *algebra.DropScope) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitCreateCollection(stmt *algebra.CreateCollection) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitDropCollection(stmt *algebra.DropCollection) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitFlushCollection(stmt *algebra.FlushCollection) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}
