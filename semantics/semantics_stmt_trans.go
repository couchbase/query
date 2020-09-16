//  Copyright (c) 2020 Couchbase, Inc.
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
	"github.com/couchbase/query/errors"
)

func (this *SemChecker) VisitStartTransaction(stmt *algebra.StartTransaction) (interface{}, error) {
	if !this.hasSemFlag(_SEM_ENTERPRISE) {
		return nil, errors.NewEnterpriseFeature("Transactions", "semantics.visit_transactions")
	}
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitCommitTransaction(stmt *algebra.CommitTransaction) (interface{}, error) {
	if !this.hasSemFlag(_SEM_ENTERPRISE) {
		return nil, errors.NewEnterpriseFeature("Transactions", "semantics.visit_transactions")
	}
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitRollbackTransaction(stmt *algebra.RollbackTransaction) (interface{}, error) {
	if !this.hasSemFlag(_SEM_ENTERPRISE) {
		return nil, errors.NewEnterpriseFeature("Transactions", "semantics.visit_transactions")
	}
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitTransactionIsolation(stmt *algebra.TransactionIsolation) (interface{}, error) {
	if !this.hasSemFlag(_SEM_ENTERPRISE) {
		return nil, errors.NewEnterpriseFeature("Transactions", "semantics.visit_transactions")
	}
	return nil, stmt.MapExpressions(this)
}

func (this *SemChecker) VisitSavepoint(stmt *algebra.Savepoint) (interface{}, error) {
	if !this.hasSemFlag(_SEM_ENTERPRISE) {
		return nil, errors.NewEnterpriseFeature("Transactions", "semantics.visit_transactions")
	}
	return nil, stmt.MapExpressions(this)
}
