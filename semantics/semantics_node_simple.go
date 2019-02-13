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
	"github.com/couchbase/query/errors"
)

func (this *SemChecker) VisitKeyspaceTerm(node *algebra.KeyspaceTerm) (interface{}, error) {
	return nil, node.MapExpressions(this)
}

func (this *SemChecker) VisitExpressionTerm(node *algebra.ExpressionTerm) (interface{}, error) {
	if node.IsKeyspace() {
		return node.KeyspaceTerm().Accept(this)
	}

	return node.ExpressionTerm().Accept(this)
}

func (this *SemChecker) VisitSubqueryTerm(node *algebra.SubqueryTerm) (interface{}, error) {
	if node.Subquery().IsCorrelated() {
		return nil, errors.NewNoCorrelatedSubqTermError(node.Alias())
	}
	return node.Subquery().Accept(this)
}
