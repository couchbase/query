//  Copyright (c) 2019 Couchbase, Inc.
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

func (this *Rewrite) VisitUpdateStatistics(stmt *algebra.UpdateStatistics) (interface{}, error) {
	return stmt, stmt.MapExpressions(this)
}
