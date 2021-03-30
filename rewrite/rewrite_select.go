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
	"github.com/couchbase/query/expression"
)

func (this *Rewrite) VisitSelectTerm(node *algebra.SelectTerm) (interface{}, error) {
	return node.Select().Accept(this)
}

func (this *Rewrite) VisitSubselect(node *algebra.Subselect) (r interface{}, err error) {
	return node, node.MapExpressions(this)
}

func (this *Rewrite) VisitSubquery(expr expression.Subquery) (r interface{}, err error) {
	if node, ok := expr.(*algebra.Subquery); ok {
		_, err = node.Select().Accept(this)
	}
	return expr, err
}
