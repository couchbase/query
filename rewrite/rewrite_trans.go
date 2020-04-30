//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package rewrite

import (
	"github.com/couchbase/query/algebra"
)

func (this *Rewrite) VisitStartTransaction(stmt *algebra.StartTransaction) (interface{}, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitCommitTransaction(stmt *algebra.CommitTransaction) (interface{}, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitRollbackTransaction(stmt *algebra.RollbackTransaction) (interface{}, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitTransactionIsolation(stmt *algebra.TransactionIsolation) (interface{}, error) {
	return stmt, stmt.MapExpressions(this)
}

func (this *Rewrite) VisitSavepoint(stmt *algebra.Savepoint) (interface{}, error) {
	return stmt, stmt.MapExpressions(this)
}
