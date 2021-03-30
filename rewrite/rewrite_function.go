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

func (this *Rewrite) VisitFunction(expr expression.Function) (interface{}, error) {
	if agg, ok := expr.(algebra.Aggregate); ok {
		if err := this.visitAggregateFunction(agg); err != nil {
			return expr, err
		}
	}

	return expr, expr.MapChildren(this)
}

func (this *Rewrite) visitAggregateFunction(agg algebra.Aggregate) (err error) {
	if this.hasRewriteFlag(REWRITE_PHASE1) {
		wTerm := agg.WindowTerm()
		if wTerm != nil {
			err = wTerm.RewriteToNewWindowTerm(this.windowTerms)
		}
	}
	return err
}
