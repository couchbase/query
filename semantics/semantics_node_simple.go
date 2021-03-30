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
	return node.Subquery().Accept(this)
}
