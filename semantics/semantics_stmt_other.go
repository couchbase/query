//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package semantics

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
)

func (this *SemChecker) VisitGrantRole(stmt *algebra.GrantRole) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitRevokeRole(stmt *algebra.RevokeRole) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitExplain(stmt *algebra.Explain) (interface{}, error) {
	return stmt.Statement().Accept(this)
}

func (this *SemChecker) VisitAdvise(stmt *algebra.Advise) (interface{}, error) {
	if !this.hasSemFlag(_SEM_ENTERPRISE) {
		return nil, errors.NewEnterpriseFeature("Advise", "semantics.visit_advise")
	}
	switch stmt.Statement().Type() {
	case "SELECT", "DELETE", "MERGE", "UPDATE":
		return stmt.Statement().Accept(this)
	default:
		return nil, errors.NewAdviseUnsupportedStmtError("semantics.visit_advise")
	}
}

func (this *SemChecker) VisitPrepare(stmt *algebra.Prepare) (interface{}, error) {
	return stmt.Statement().Accept(this)
}

func (this *SemChecker) VisitExecute(stmt *algebra.Execute) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitInferKeyspace(stmt *algebra.InferKeyspace) (interface{}, error) {
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitUpdateStatistics(stmt *algebra.UpdateStatistics) (interface{}, error) {
	if !this.hasSemFlag(_SEM_ENTERPRISE) {
		return nil, errors.NewEnterpriseFeature("Update Statistics", "semantics.visit_update_statistics")
	}
	if (stmt.IndexAll() || len(stmt.Indexes()) > 0) &&
		(stmt.Using() != datastore.GSI && stmt.Using() != datastore.DEFAULT) {
		return nil, errors.NewUpdateStatInvalidIndexTypeError()
	}
	if stmt.IndexAll() && !stmt.Keyspace().Path().IsCollection() {
		return nil, errors.NewUpdateStatIndexAllCollectionOnly()
	}
	return nil, stmt.MapExpressions(this)
}
